package controllers

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"odooNew/helpers"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB
var GlobalParam map[string]string

func InitDB(user string, pass string, dbName string, port string, host string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbName)
	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
		// return
	}
	// defer db.Close()
	err = DB.Ping()
	if err != nil {
		return err
	}

	fmt.Println("Successfully connected to the database!")
	return nil
}

func InitParam(data map[string]string) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	GlobalParam = data
	fmt.Println(GlobalParam)
}

func FullInput(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	xTask := r.Header.Get("X-Task")
	if xTask == "" {
		http.Error(w, "TASK ??", http.StatusMethodNotAllowed)
		return
	}
	userLogin := r.Header.Get("login")
	passLogin := r.Header.Get("passlogin")
	if userLogin == "" || passLogin == "" {
		userLogin = GlobalParam["LOGIN"]
		passLogin = GlobalParam["PASS"]
	}

	xTask = helpers.SafeHeaderValue(xTask)
	key := r.Header.Get("X-Anaconda")
	if key != GlobalParam["KEY"] {
		http.Error(w, "NO", http.StatusBadRequest)
		return
	}
	mainPath := GlobalParam["PATH_FULL"]
	// fmt.Println(mainPath, "bet", xTask)

	err := r.ParseMultipartForm(100 << 20) // 100 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	slice1 := strings.Split(GlobalParam["ArrayPicture"], ",")
	// fmt.Println("disini", slice1)
	err = os.MkdirAll(fmt.Sprintf("%s/%s", mainPath, xTask), os.ModePerm)
	if err != nil {
		http.Error(w, "Failed to create source directory", http.StatusInternalServerError)
		return
	}
	for _, field := range slice1 {
		file, header, err := r.FormFile(field)
		if err != nil {
			// Skip fields without uploaded files

			continue
		}
		defer file.Close()

		// Detect the content type based on the file content
		if !strings.HasPrefix(header.Header.Get("Content-Type"), "image/") {

			continue
		}

		dst, err := os.Create(fmt.Sprintf("%s/%s/%s.jpg", mainPath, xTask, field))
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the destination
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Error saving file", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
		// dataFoto += field + ","
	}
	file, _, err := r.FormFile("json")
	if err != nil {

	} else {
		dst, err := os.Create(fmt.Sprintf("%s/%s/%s.json", mainPath, xTask, "file"))
		if err != nil {
			http.Error(w, "Error saving file json", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the destination
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Error saving file json", http.StatusInternalServerError)
			return
		}
	}
	defer file.Close()
	xSubmit := r.Header.Get("X-Submit")
	if xSubmit == "1" {
		pathReady := fmt.Sprintf("%s/%s/readyBro", GlobalParam["PATH_FULL"], xTask)
		err := helpers.Touch(pathReady)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res, code, err := helpers.FullOdoo(GlobalParam["linkUpdate"], xTask, userLogin, passLogin, GlobalParam, DB)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(code)
		w.Write([]byte(res))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File submit successfully"))

}

func RefreshIT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

}

func GetData(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userLogin := r.Header.Get("login")
	passLogin := r.Header.Get("passlogin")
	if userLogin == "" || passLogin == "" {
		userLogin = GlobalParam["LOGIN"]
		passLogin = GlobalParam["PASS"]
	}

	key := r.Header.Get("X-Anaconda")
	if key != GlobalParam["KEY"] {
		http.Error(w, "NO", http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	coo, err1 := helpers.PostLoginOdoo(GlobalParam["linkLogin"], userLogin, passLogin)
	if err1 != nil {
		fmt.Println(err1.Error())
		return
	}
	isCheck := false
	var data struct {
		Params struct {
			Model string `json:"model"`
		} `json:"params"`
	}
	err5 := json.Unmarshal(body, &data)
	if err5 != nil {
		// fmt.Println("Error parsing JSON:", err)
		// return
	} else {
		if data.Params.Model == "project.task" {
			isCheck = true
		}
	}
	ret, err8 := helpers.PostPassOdoo(GlobalParam["linkGet"], body, coo)

	if err8 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err8.Error()))
	}

	if isCheck {
		var ids []string
		type ResponseGet struct {
			JSONRPC string                   `json:"jsonrpc"`
			ID      *int                     `json:"id"`
			Result  []map[string]interface{} `json:"result"`
		}

		var response ResponseGet
		err4 := json.Unmarshal([]byte(ret), &response)
		if err4 != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(ret))
			return

		} else {

		}
		for _, task := range response.Result {
			ids = append(ids, fmt.Sprintf("%d", int(task["id"].(float64))))
		}
		existingIDs, err := helpers.GetExistingIDs(ids, GlobalParam["PATH_FULL"])
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(ret))
			return
		}
		for i := 0; i < len(response.Result); i++ {
			if existingIDs[fmt.Sprintf("%d", int(response.Result[i]["id"].(float64)))] == true {
				response.Result = append(response.Result[:i], response.Result[i+1:]...)
			}
		}
		updatedJSON, err := json.Marshal(response)
		if err != nil {
			fmt.Println("Error converting to JSON:", err)
			w.WriteHeader(500)
			w.Write([]byte(ret))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(updatedJSON)
		return
	} else {
		// )
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ret))
		return

	}
	// w.WriteHeader(http.StatusOK)
	// w.Write([]byte("File submit successfully"))

}
