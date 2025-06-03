package helpers

// package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func Touch(filePath string) error {
	currentTime := time.Now()

	// Try to open the file (create it if it doesn't exist)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Update the access and modification times to now
	return os.Chtimes(filePath, currentTime, currentTime)
}

// CopyFile copies a file from src to dstDir.
// It creates dstDir if it doesn't exist.

func CopyFile(src, dstDir string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get the file name and create destination file path
	dstPath := filepath.Join(dstDir, filepath.Base(src))
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy contents from source to destination
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
func insertLog(db *sql.DB, idTask string, status int, body string, typeSend string) error {
	query := `INSERT INTO log_full_task (id_task, response, status_code,type_send) VALUES (?, ?, ?,?)`

	_, err := db.Exec(query, idTask, body, status, typeSend)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
func MoveFolder(srcDir, dstDir string) error {
	// Make sure the source exists
	srcInfo, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("source directory does not exist: %w", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	// Create the destination directory
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Walk through the source directory
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute the destination path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			// Create sub-directory
			return os.MkdirAll(dstPath, info.Mode())
		} else {
			// Copy the file
			return copyFile(path, dstPath)
		}
	})
	if err != nil {
		return fmt.Errorf("error copying folder: %w", err)
	}

	// Remove the original directory after copying
	return os.RemoveAll(srcDir)
}

func LoadJSONToMap(filename string) (map[string]interface{}, map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal membuka file: %w", err)
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal membaca isi file: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(byteValue, &result); err != nil {
		return nil, nil, fmt.Errorf("gagal unmarshal JSON: %w", err)
	}
	var result1 map[string]interface{}
	if err := json.Unmarshal(byteValue, &result1); err != nil {
		return nil, nil, fmt.Errorf("gagal unmarshal JSON: %w", err)
	}

	return result, result1, nil
}
func getSecondArray(data map[string]interface{}, key string) string {
	arr := checkArray(data[key])
	if len(arr) > 1 {
		return checkString(arr[1])
	}
	return "tanpa nama teknisi"
}
func removeFalsyValues(data map[string]interface{}) {
	for k, v := range data {
		if k == "stage_id" {
			data[k] = 5
		}

		val1 := reflect.TypeOf(v)
		val := reflect.ValueOf(v)
		if val1.Kind() == reflect.Slice && val.Len() > 0 {
			// fmt.Println(k)
			firstElem := val.Index(0).Interface() // Get the first element
			// fmt.Println(firstElem)
			// Check if the first element is an int before assigning it to v
			// if num, ok := firstElem.(int); ok {

			data[k] = firstElem
			// }
		}

		switch value := v.(type) {
		case map[string]interface{}:
			removeFalsyValues(value) // Recursive call for nested maps
			if len(value) == 0 {
				delete(data, k) // Delete the key if the map is empty
			}
		case bool:
			if !value {
				delete(data, k) // Delete the key if the value is false
			}
		case string:
			if value == "" {
				delete(data, k) // Delete the key if the value is an empty string
			}
		case float64:
			if value == 0 {
				delete(data, k) // Delete the key if the value is 0
			}
			fmt.Println("yo->", k)
		}
	}
}

// Helper: copy individual file
func copyFile(src, dst string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return err
}

func SafeHeaderValue(header string) string {
	// 1. Remove dangerous path components
	header = filepath.Base(header) // removes ../ and directories

	// 2. Remove unwanted characters (allow only letters, digits, -, _, and .)
	safeChars := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	header = safeChars.ReplaceAllString(header, "")

	// 3. Limit length
	if len(header) > 100 {
		header = header[:100]
	}

	return header
}

func checkString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func FolderExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}

func GetExistingIDs(ids []string, root string) (map[string]bool, error) {
	// Convert IDs to SQL IN clause format

	// Store found IDs in a map
	existingIDs := make(map[string]bool)
	for i := 0; i < len(ids); i++ {
		path := fmt.Sprintf("%s/%s/readyBro", root, ids[i])
		if !fileExists(path) {
			existingIDs[ids[i]] = true
		}
	}

	return existingIDs, nil
}

func checkStringTanpa(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "Tanpa Data"
}

func checkBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func stringToMap(s string) map[string]int {
	result := make(map[string]int)
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result[part] = 1
		}
	}
	return result
}

func checkInt(v interface{}) int {
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

func checkArray(v interface{}) []any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		result := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = val.Index(i).Interface()
		}
		return result
	}
	return []any{}
}
func getDataPosition(db *sql.DB, partnerID int) (string, string, error) {
	var lat, long string

	query := `SELECT lat, long FROM position_partner WHERE noReal = ? LIMIT 1`
	err := db.QueryRow(query, partnerID).Scan(&lat, &long)
	if err != nil {
		return "", "", fmt.Errorf("query error: %v", err)
	}

	return lat, long, nil
}

func checkPhoto(arrayCheck string, idTask string, root string) (bool, error) {
	loadCheck := stringToMap(arrayCheck)
	mainPath := fmt.Sprintf("%s/%s/", root, idTask)
	for key, _ := range loadCheck {
		if !fileExists(mainPath + key + ".jpg") {
			return false, fmt.Errorf("not exist ->", key)
		}
	}
	return true, nil
}

func cleanStringSource(input string) string {
	// Remove unwanted words
	cleaned := strings.ReplaceAll(input, `"SHARING"`, "")
	cleaned = strings.ReplaceAll(cleaned, `"SINGLE"`, "")
	cleaned = strings.ReplaceAll(cleaned, `"`, "") // remove remaining quotes
	// Replace hyphens with commas
	cleaned = strings.ReplaceAll(cleaned, "-", ",")
	// Optionally, trim extra spaces
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

func checkPhotoDeep(idTask string, root string, x_source string, ceasheet bool) (bool, error) {
	// loadCheck := stringToMap(arrayCheck)
	x_source = cleanStringSource(x_source)
	mapSource := stringToMap(x_source)
	mainPath := fmt.Sprintf("%s/%s/", root, idTask)
	if !fileExists(mainPath + "x_foto_sticker_edc" + ".jpg") {
		return false, fmt.Errorf("not exist ->", "x_foto_sticker_edc")
	}
	if !fileExists(mainPath + "x_foto_screen_guard" + ".jpg") {
		return false, fmt.Errorf("not exist ->", "x_foto_screen_guard")
	}
	if !fileExists(mainPath + "x_foto_all_transaction" + ".jpg") {
		return false, fmt.Errorf("not exist ->", "x_foto_all_transaction")
	}
	if !fileExists(mainPath + "x_foto_transaksi_patch" + ".jpg") {
		if !fileExists(mainPath + "x_foto_screen_p2g" + ".jpg") {
			return false, fmt.Errorf("not exist ->", "x_foto_transaksi_path AND x_foto_screen_p2g")
		}
	}
	if ceasheet {
		if !fileExists(mainPath + "x_foto_telp_pic_belakang_edc" + ".jpg") {
			return false, fmt.Errorf("not exist ->", "x_foto_telp_pic_belakang_edc")
		}
	}
	if mapSource["BMRI"] == 1 {
		if !fileExists(mainPath + "x_foto_transaksi_bmri" + ".jpg") {
			return false, fmt.Errorf("not exist ->", "x_foto_transaksi_bmri")
		}

	}
	if mapSource["BNI"] == 1 {
		if !fileExists(mainPath + "x_foto_transaksi_bni" + ".jpg") {
			return false, fmt.Errorf("not exist ->", "x_foto_transaksi_bni")
		}

	}
	if mapSource["BRI"] == 1 {
		if !fileExists(mainPath + "x_foto_transaksi_bri" + ".jpg") {
			return false, fmt.Errorf("not exist ->", "x_foto_transaksi_bri")
		}

	}
	if mapSource["BTN"] == 1 {
		if !fileExists(mainPath + "x_foto_transaksi_btn" + ".jpg") {
			return false, fmt.Errorf("not exist ->", "x_foto_transaksi_btn")
		}

	}

	return true, nil
}

func insertPhotoJson(data map[string]interface{}, dataPicture string, root string, idTask string) (string, error) {
	// Get the existing "params" map
	mainPath := fmt.Sprintf("%s/%s/", root, idTask)
	if params, ok := data["params"].(map[string]interface{}); ok {
		mapPicture := stringToMap(dataPicture)
		for key, _ := range mapPicture {
			path := mainPath + key + ".jpg"
			if fileExists(path) {
				dataBase, err := fileToBase64(path)
				if err != nil {
					return "", err
				}
				params[key] = dataBase
			}
		}
		// Add new key-value pairs to "params"

	} else {

		return "", fmt.Errorf("JSON NO VALID")
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("JSON NO VALID")
	}

	return string(jsonBytes), nil
}

func sanitizeIt(data string) string {
	data = strings.ReplaceAll(data, "\r", " ")
	data = strings.ReplaceAll(data, "\n", " ")
	data = strings.ReplaceAll(data, "{", "")
	data = strings.ReplaceAll(data, "}", "")
	data = strings.ReplaceAll(data, "[", "")
	data = strings.ReplaceAll(data, "]", "")
	data = strings.ReplaceAll(data, `"`, "")
	ret := strings.TrimSpace(data)
	return ret

}
func insertPhotoJsonViaString(dataString string, dataPicture string, root string, idTask string) (string, error) {
	// Get the existing "params" map
	data, err := stringToMapInterface(dataString)
	if err != nil {
		return "", err
	}

	mainPath := fmt.Sprintf("%s/%s/", root, idTask)
	if params, ok := data["params"].(map[string]interface{}); ok {
		mapPicture := stringToMap(dataPicture)
		for key, _ := range mapPicture {
			path := mainPath + key + ".jpg"
			if fileExists(path) {
				dataBase, err := fileToBase64(path)
				if err != nil {
					return "", err
				}
				params[key] = dataBase
			}
		}
		// Add new key-value pairs to "params"

	} else {

		return "", fmt.Errorf("JSON NO VALID")
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("JSON NO VALID")
	}

	return string(jsonBytes), nil
}
func stringToMapInterface(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}
func fileToBase64(path string) (string, error) {
	// Read file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || !os.IsNotExist(err)
}
func checkDistanceFull(srcLong string, srcLat string, dataLat string, dataLong string) int {
	floatLong, err := strconv.ParseFloat(srcLong, 64)
	if err != nil {
		// w.WriteHeader(http.StatusBadRequest)
		// w.Write([]byte("Bad Body :("))
		fmt.Println("a")
		return 1
	}
	floatLat, err := strconv.ParseFloat(srcLat, 64)
	if err != nil {
		// w.WriteHeader(http.StatusBadRequest)
		// w.Write([]byte("Bad Body :("))
		fmt.Println("b")
		return 1
	}
	hh := 9
	if hh == 1 {

	} else {
		if dataLong == "0" || dataLat == "0" {

		} else {
			floatLongOld, errlong := strconv.ParseFloat(dataLong, 64)
			// if err != nil {
			// 	// w.WriteHeader(http.StatusBadRequest)
			// 	// w.Write([]byte("Bad Body :("))
			// 	// return
			if errlong != nil {
				fmt.Println(errlong)
				fmt.Println("o")

				return 1
			}

			// }
			floatLatOld, errlat := strconv.ParseFloat(dataLat, 64)
			// if err != nil {
			// 	// w.WriteHeader(http.StatusBadRequest)
			// 	// w.Write([]byte("Bad Body :("))
			// 	// return
			// }
			if errlat != nil {
				// fmt.Println(errlat)
				fmt.Println("p")
				return 1
			}
			if errlat == nil && errlong == nil {
				if floatLongOld == 0 && floatLatOld == 0 {

				} else {
					if checkDistance(floatLatOld, floatLongOld, floatLat, floatLong) {
						// flagAIBro = 2
						// w.WriteHeader(http.StatusBadRequest)
						// w.Write([]byte("Pengerjaan Jarak Terlalu Jauh, Maksimal 100 Meter"))
						// return
						return 2
					} else {
						return 3
					}
				}

			}

		}
	}
	return 3
}

func extractInt(input interface{}) int {
	fmt.Printf("Actual type: %T, value: %#v\n", input, input)
	switch v := input.(type) {
	case []interface{}:
		if len(v) > 0 {
			if i, ok := v[0].(int); ok {
				return i
			}
		}
	case int:
		return v
	case float64:
		return int(v)

	}

	return 0 // default fallback if not found or not int
}

func checkDistance(lat1, lon1, lat2, lon2 float64) bool {
	distance := haversine(lat1, lon1, lat2, lon2)
	fmt.Println("lat1", lat1, "lon1", lon1, "lat2", lat2, "lon2", lon2, "distance", distance)
	return distance > 100 // return true if distance is more than 100 meters
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371e3 // Earth's radius in meters

	// Convert latitude and longitude from degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distance in meters
	distance := earthRadius * c
	return distance
}
