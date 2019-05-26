package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	xrdb map[string]string
)

type color struct {
	rgbValue string
	hexValue string
}

// Color stores hex and rgb value of color
type Color interface {
	Hex() string
	RGB() string
}

// ParseColor parses a string in both hex or rgb
// or from XResources or env variable
// and converts to both rgb and hex value
func ParseColor(raw string) Color {
	var red int64
	var green int64
	var blue int64

	if strings.HasPrefix(raw, "${") {
		endIndex := len(raw) - 1
		raw = raw[2:endIndex]

		// From XResources database
		if strings.HasPrefix(raw, "xrdb:") {
			raw = fromXResources(raw)

			// From environment variable
		} else if env := os.Getenv(raw); len(env) > 0 {
			raw = env
		}
	}

	// rrr,bbb,ggg
	if strings.Contains(raw, ",") {
		list := strings.SplitN(raw, ",", 3)
		list = append(list, "255", "255")

		red = stringToInt(list[0], 10)
		green = stringToInt(list[1], 10)
		blue = stringToInt(list[2], 10)

	} else {
		re := regexp.MustCompile("[a-fA-F0-9]+")
		hex := re.FindString(raw)

		// Support short hex color form e.g. #fff, #121
		if len(hex) == 3 {
			expanded := []byte{
				hex[0], hex[0],
				hex[1], hex[1],
				hex[2], hex[2]}

			hex = string(expanded)
		}

		hex += "ffffff"

		red = stringToInt(hex[:2], 16)
		green = stringToInt(hex[2:4], 16)
		blue = stringToInt(hex[4:6], 16)
	}

	return color{
		rgbValue: fmt.Sprintf("%d,%d,%d", red, green, blue),
		hexValue: fmt.Sprintf("%02x%02x%02x", red, green, blue)}
}

func (c color) Hex() string {
	return c.hexValue
}

func (c color) RGB() string {
	return c.rgbValue
}

func stringToInt(raw string, base int) int64 {
	value, err := strconv.ParseInt(raw, base, 0)
	if err != nil {
		value = 255
	}

	if value < 0 {
		value = 0
	}

	if value > 255 {
		value = 255
	}

	return value
}

func getXRDB() error {
	db := map[string]string{}

	if len(xrdb) > 0 {
		return nil
	}

	output, err := exec.Command("xrdb", "-query").Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	re := regexp.MustCompile(`^\*\.?(\w+?):\s*?#([A-Za-z0-9]+)`)
	for scanner.Scan() {
		line := scanner.Text()
		for _, match := range re.FindAllStringSubmatch(line, -1) {
			if match != nil {
				db[match[1]] = match[2]
			}
		}
	}

	xrdb = db

	return nil
}

func fromXResources(input string) string {
	// Example input:
	//   xrdb:color1
	//   xrdb:color2:#f0c
	//   xrdb:color5:40,50,60
	queries := strings.Split(input, ":")
	if len(queries[1]) == 0 {
		PrintError(`"` + input + `": Wrong XResources lookup syntax`)
		os.Exit(0)
	}

	if err := getXRDB(); err != nil {
		Fatal(err)
	}

	if len(xrdb) < 1 {
		PrintError("XResources is not available")
		os.Exit(0)
	}

	value, ok := xrdb[queries[1]]

	if !ok || len(value) == 0 {
		if len(queries) > 2 {
			// Fallback value
			value = queries[2]
		} else {
			PrintError("Variable is not available in XResources")
			os.Exit(0)
		}
	}

	return value
}
