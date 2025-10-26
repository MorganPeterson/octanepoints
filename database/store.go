package database

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/parser"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	_ "modernc.org/sqlite"
)

type CarsWrapper struct {
	Cars []Cars `json:"cars"`
}

// Store wraps your GORM DB instance.
type Store struct {
	DB *gorm.DB
}

// NewStore opens (or creates) the SQLite file at path, applies
// connection settings, and runs migrations.
func NewStore(path string) (*Store, error) {
	// Open with a bit of GORM logging enabled; adjust logger level if needed.
	gormDB, err := gorm.Open(
		sqlite.Dialector{
			DSN:        path + "?_foreign_keys=on",
			DriverName: "sqlite",
		},
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm sqlite: %w", err)
	}

	// Grab the underlying *sql.DB to set connection limits.
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("getting sql.DB from gorm: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Minute)

	store := &Store{DB: gormDB}
	if err := store.Migrate(); err != nil {
		return nil, fmt.Errorf("automigrate failed: %w", err)
	}

	return store, nil
}

// Migrate runs AutoMigrate on all your models.
func (s *Store) Migrate() error {
	if err := s.DB.AutoMigrate(
		&RallyOverall{},
		&RallyStage{},
		&Rally{},
		&Cars{},
		&Class{},
		&ClassCar{},
		&ClassDriver{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	var count int64
	if err := s.DB.Model(&Cars{}).Count(&count).Error; err != nil {
		return fmt.Errorf("counting cars: %w", err)
	}
	if count == 0 {
		if err := seedCarsAndClasses(s.DB, "cars.json"); err != nil {
			return fmt.Errorf("seeding cars: %w", err)
		}
	}

	err := seedClassesAndMembers(s.DB, configuration.MustLoad("config.toml"))
	if err != nil {
		return fmt.Errorf("seeding classes and members: %w", err)
	}

	return nil
}

// Close cleanly shuts down the database connection.
func (s *Store) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func seedClassesAndMembers(db *gorm.DB, config *configuration.Config) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// upsert all classes
		cls := make([]Class, len(config.Classes))
		for i, cc := range config.Classes {
			cls[i] = Class{
				Name:        cc.Name,
				Slug:        parser.Slugify(cc.Name),
				Description: cc.Description,
				Active:      true,
			}
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoNothing: true,
		}).Create(&cls).Error; err != nil {
			return fmt.Errorf("upserting classes: %w", err)
		}

		// Build a map[slug]classID & a map[name]slug
		var dbCls []Class
		slugs := make([]string, 0, len(config.Classes))
		nameToSlug := make(map[string]string, len(config.Classes))
		for _, c := range config.Classes {
			slug := parser.Slugify(c.Name)
			nameToSlug[c.Name] = slug
			slugs = append(slugs, slug)
		}

		if err := tx.Where("slug IN ?", slugs).Find(&dbCls).Error; err != nil {
			return fmt.Errorf("finding classes: %w", err)
		}

		slugToID := make(map[string]int64, len(dbCls))
		for _, c := range dbCls {
			slugToID[nameToSlug[c.Name]] = c.ID
		}

		// for each class, upsert ClassCar rows
		var catJoins []ClassCar
		categories := []string{}
		for _, c := range config.Classes {
			for _, cat := range c.Categories {
				categories = append(categories, cat) // collect unique categories
			}
		}

		// find all cars in that category
		var cars []Cars
		if err := tx.Where("category IN ?", categories).Find(&cars).Error; err != nil {
			return fmt.Errorf("finding cars: %w", err)
		}

		for _, car := range cars {
			catJoins = append(catJoins, ClassCar{
				ClassID: slugToID[nameToSlug[car.Category]],
				CarID:   car.ID,
			})
		}
		if len(catJoins) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "class_id"}, {Name: "car_id"}},
				DoNothing: true,
			}).Create(&catJoins).Error; err != nil {
				return fmt.Errorf("upserting class-car joins: %w", err)
			}
		}

		// upsert ClassDriver rows
		var drvJoins []ClassDriver
		for _, c := range config.Classes {
			cid := slugToID[nameToSlug[c.Name]]
			for _, uname := range c.Drivers {
				// look up UserId from your last import (RallyOverall table)
				var ro RallyOverall
				if err := tx.Select("user_id").
					Where("user_name = ?", uname).
					Order("rally_id DESC").
					First(&ro).Error; err != nil {
					continue // skip if unknown driver
				}
				drvJoins = append(drvJoins, ClassDriver{
					ClassID: cid,
					UserId:  ro.UserId,
				})
			}
		}
		if len(drvJoins) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "class_id"}, {Name: "user_id"}},
				DoNothing: true,
			}).Create(&drvJoins).Error; err != nil {
				return fmt.Errorf("upserting class-driver joins: %w", err)
			}
		}
		return nil
	})
}

// seedFromJSON reads a JSON file and uses that data to seed the Cars and Class
// related tables. It assumes the JSON structure matches the Cars model.
func seedCarsAndClasses(db *gorm.DB, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var wrapper CarsWrapper
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return err
	}
	if len(wrapper.Cars) == 0 {
		return nil
	}

	// slugify car names and ensure they are unique
	for i := range wrapper.Cars {
		wrapper.Cars[i].Slug = parser.Slugify(wrapper.Cars[i].Brand + " " + wrapper.Cars[i].Model)
		if wrapper.Cars[i].Slug == "" {
			return fmt.Errorf("car slug is empty for car: %s %s", wrapper.Cars[i].Brand, wrapper.Cars[i].Model)
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 1) Insert cars (if empty)
		var carCount int64
		if err := tx.Model(&Cars{}).Count(&carCount).Error; err != nil {
			return err
		}
		if carCount == 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "rsf_id"}},
				DoNothing: true,
			}).Create(&wrapper.Cars).Error; err != nil {
				return err
			}
		} else {
			// Ensure IDs are loaded if they already exist
			for i := range wrapper.Cars {
				// var id int64
				var car Cars
				if err := tx.Where("rsf_id = ?", wrapper.Cars[i].RSFID).First(&car).Error; err != nil {
					return err
				}
				wrapper.Cars[i].ID = car.ID
			}
		}

		// 2) Build distinct classes from car.Category
		type tmpClass struct {
			Name string
			Slug string
		}
		seen := map[string]struct{}{}
		classesToInsert := make([]Class, 0)
		for _, car := range wrapper.Cars {
			if _, ok := seen[car.Category]; ok {
				continue
			}
			seen[car.Category] = struct{}{}
			classesToInsert = append(classesToInsert, Class{
				Name:        car.Category,
				Slug:        parser.Slugify(car.Category),
				Description: "",
				Active:      true,
			})
		}

		// 3) Upsert classes (ignore if exists)
		if len(classesToInsert) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "slug"}},
				DoNothing: true,
			}).Create(&classesToInsert).Error; err != nil {
				return err
			}
		}

		// Get ids for all categories we saw
		var dbClasses []Class
		slugs := make([]string, 0, len(seen))
		for k := range seen {
			slugs = append(slugs, parser.Slugify(k))
		}
		if err := tx.Where("slug IN ?", slugs).Find(&dbClasses).Error; err != nil {
			return err
		}
		nameToID := make(map[string]int64, len(dbClasses))
		for _, c := range dbClasses {
			nameToID[c.Name] = c.ID
		}

		// 4) Build ClassCar join rows
		ccRows := make([]ClassCar, 0, len(wrapper.Cars))
		for _, car := range wrapper.Cars {
			classID := nameToID[car.Category]
			ccRows = append(ccRows, ClassCar{
				ClassID: classID,
				CarID:   car.ID,
			})
		}

		// Upsert/ignore duplicates
		if len(ccRows) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "class_id"}, {Name: "car_id"}},
				DoNothing: true,
			}).Create(&ccRows).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
