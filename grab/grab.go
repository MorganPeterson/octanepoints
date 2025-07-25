package grab

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

const rallyCSVURLTmpl = "https://rallysimfans.hu/rbr/csv_export_beta.php?rally_id=%d"
const rallyCSVOverallTmpl = "https://rallysimfans.hu/rbr/csv_export_results.php?rally_id=%d&cg=7"
const rallyDir = "rallies"
const stageFileName = "_table.csv"
const overallFileName = "_All_table.csv"

type Paths struct {
	Id   int64  // rally ID
	Dir  string // directory where the file is saved
	TOML string // path to the TOML file
}

func Grab(ctx context.Context, id int64) error {
	p, err := prepare(id)
	if err != nil {
		return fmt.Errorf("failed to prepare paths: %w", err)
	}

	// download the stages results
	if _, err := stagesDownload(ctx, p); err != nil {
		return fmt.Errorf("failed to download stages results: %w", err)
	}

	// download the overall results
	if _, err := overallDownload(ctx, p); err != nil {
		return fmt.Errorf("failed to download overall results: %w", err)
	}

	return nil
}

// download downloads a file from the specified URL and saves it to outPath.
func download(ctx context.Context, rawUrl string, outPath string) error {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return err
	}

	// derive a UTF-8 filename from the URL if caller didn't supply one
	if outPath == "" {
		base := filepath.Base(u.Path)
		if base == "" || base == "/" {
			base = "downloaded_file"
		}
		if u.RawQuery != "" {
			base += "?" + u.RawQuery
		}
		name, err := url.PathUnescape(base) // ensure UTF-8 encoding
		if err != nil {
			// fallback
			name = base
		}
		outPath = name
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// If HTTPS, wrap with TLS using system CAs.
	var rw io.ReadWriter = conn
	if u.Scheme == "https" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("failed to load system cert pool: %w", err)
		}
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: host, // SNI + verify
			RootCAs:    pool,
		})

		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("tls handshake: %w", err)
		}

		rw = tlsConn
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawUrl, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Header = http.Header{
		"User-Agent":      []string{"Wget/1.25.0"}, // fake it until you make it
		"Accept":          []string{"*/*"},
		"Accept-Encoding": []string{"identity"},
		"Connection":      []string{"Keep-Alive"},
	}
	req.Host = host

	bw := bufio.NewWriter(rw)
	if err := req.Write(bw); err != nil {
		return fmt.Errorf("write request: %w", err)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush request: %w", err)
	}

	// read the response
	br := bufio.NewReader(rw)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		// drain to avoid leaking the connection if re-used
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf(("bad status code %d for %s"), resp.StatusCode, string(b))
	}

	// save body to file
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}

func encodeRallyToml() []byte {
	var buf = new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(map[string]any{
		"rally": map[string]any{
			"rallyId":          0,
			"name":             "rally name",
			"description":      "description of rally",
			"creator":          "John Doe",
			"damageLevel":      "reduced",
			"numberOfLegs":     3,
			"superRally":       true,
			"pacenotesOptions": "Normal Pacenotes",
			"started":          0,
			"finished":         0,
			"totalDistance":    0.0,
			"carGroups":        "Group A8, Group A7",
			"startAt":          "2025-06-24 08:00",
			"endAt":            "2025-07-01 08:00",
		},
	}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func overallDownload(ctx context.Context, p Paths) (string, error) {
	// download the overall results
	downloadPath := filepath.Join(p.Dir, fmt.Sprintf("%d%s", p.Id, overallFileName))

	rawUrl := fmt.Sprintf(rallyCSVOverallTmpl, p.Id)
	if err := download(ctx, rawUrl, downloadPath); err != nil {
		return downloadPath, fmt.Errorf("failed to grab %s: %w", rawUrl, err)
	}

	return downloadPath, nil
}

func prepare(id int64) (Paths, error) {
	p := Paths{Id: id}

	p.Dir = filepath.Clean(filepath.Join(rallyDir, fmt.Sprintf("%d", p.Id)))
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return p, fmt.Errorf("failed to create directory %s: %w", p.Dir, err)
	}

	p.TOML = filepath.Join(p.Dir, fmt.Sprintf("%d.toml", p.Id))
	if err := os.WriteFile(p.TOML, encodeRallyToml(), 0o644); err != nil {
		return p, fmt.Errorf("failed to create TOML file %s: %w", p.TOML, err)
	}

	return p, nil
}

func stagesDownload(ctx context.Context, p Paths) (string, error) {
	// download the stages results
	downloadPath := filepath.Join(p.Dir, fmt.Sprintf("%d%s", p.Id, stageFileName))

	rawUrl := fmt.Sprintf(rallyCSVURLTmpl, p.Id)
	if err := download(ctx, rawUrl, downloadPath); err != nil {
		return downloadPath, fmt.Errorf("failed to grab %s: %w", rawUrl, err)
	}

	return downloadPath, nil
}
