package manager

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func streamToStdout(r io.Reader) {
	if r == nil {
		return
	}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fmt.Println(sc.Text())
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func parseNames(s string) []string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 || idx+1 >= len(s) {
		return nil
	}
	trim := strings.TrimSpace(s[idx+1:])
	if trim == "" {
		return nil
	}
	parts := strings.Split(trim, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseServerProperties(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	props := make(map[string]string)
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			props[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return props, sc.Err()
}
