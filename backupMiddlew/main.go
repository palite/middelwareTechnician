package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"odooNew/controllers"
	// "github.com/joho/godotenv"
)

func main() {

	env, err := loadEnvToMap(".env")
	if err != nil {
		fmt.Println("Error loading .env:", err)
		return
	}

	envMap := make(map[string]string)
	envMap = env
	// envMap["DB_USER"]
	port := envMap["PORT"]
	dbUser := envMap["DB_USER"]
	dbPass := envMap["DB_PASS"]
	dbHost := envMap["DB_HOST"]
	dbPort := envMap["DB_PORT"]
	dbName := envMap["DB_NAME"]
	// envMap["KEY"] = os.Getenv("KEY")
	// envMap["isDistance"] = os.Getenv("isDistanceCheck")
	// envMap["linkLogin"] = os.Getenv("linkLogin")
	// envMap["linkUpdate"] = os.Getenv("linkUpdate")
	// envMap["ArrayPicture"] = os.Getenv("ArrayPicture")

	// envMap["LOGIN"] = os.Getenv("LOGIN")
	// envMap["PASS"] = os.Getenv("PASS")
	// envMap["PATH"] = os.Getenv("PATH_FULL")
	// envMap["BIN"] = os.Getenv("BIN_FULL")
	controllers.InitParam(envMap)
	err = controllers.InitDB(dbUser, dbPass, dbName, dbPort, dbHost)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer controllers.DB.Close()

	// Routing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome to Dream of your Dream! ")
	})

	http.HandleFunc("/api/getdata", controllers.GetData)

	// http.HandleFunc("/users", controllers.GetUsers)
	http.HandleFunc("/full/input", controllers.FullInput)

	fmt.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadEnvToMap(filename string) (map[string]string, error) {
	envMap := make(map[string]string)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Split KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // skip malformed lines
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove surrounding quotes if present
		value = strings.Trim(value, `"'`)
		envMap[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envMap, nil
}
