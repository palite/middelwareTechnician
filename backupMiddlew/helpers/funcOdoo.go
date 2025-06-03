package helpers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	// "odooNew/controllers"
	// "odooNew/controllers"
	// "odooNew/controllers"
)

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  struct {
		Status   int    `json:"status"`
		Success  bool   `json:"success"`
		Response bool   `json:"response"`
		Message  string `json:"message"`
	} `json:"result"`
}

type Credentials struct {
	JSONRPC string `json:"jsonrpc"`
	Params  struct {
		DB       string `json:"db"`
		Login    string `json:"login"`
		Password string `json:"password"`
	} `json:"params"`
}

func PostLoginOdoo(apiURL string, user string, pass string) ([]*http.Cookie, error) {
	credentials := Credentials{
		JSONRPC: "2.0",
		Params: struct {
			DB       string `json:"db"`
			Login    string `json:"login"`
			Password string `json:"password"`
		}{
			DB:       "gsa_db",
			Login:    user,
			Password: pass,
		},
	}
	// Convert credentials struct to JSON
	payload, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	// Send a POST request with the JSON data
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Get the cookies from the response
	cookies := resp.Cookies()

	// Example: Print out the response body
	fmt.Printf("Response Body: %s", body)

	return cookies, nil
}

func PostPassOdoo(taskURL string, data []byte, sessionIDCookie []*http.Cookie) (string, error) {
	// Convert taskData struct to JSON

	// Create a new HTTP client
	client := &http.Client{}

	// Create a cookiejar and add the session ID cookie to it
	jar, _ := cookiejar.New(nil)
	urlParsed, err := url.Parse(taskURL)
	if err != nil {
		return "", err
	}
	jar.SetCookies(urlParsed, sessionIDCookie)
	client.Jar = jar

	// Send a POST request with the JSON data and cookies
	resp, err := client.Post(taskURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Example: Print out the response body
	fmt.Printf("Response Body: %s", body)

	return string(body), nil
}

func FullOdoo(taskURL string, idTask string, login string, pass string, globalData map[string]string, db *sql.DB) (string, int, error) {
	flagDistanceAI := 1
	mainPath := globalData["PATH"]
	jsonPath := fmt.Sprintf("%s/%s/%s.json", mainPath, idTask, "file")
	rawData, pureData, err := LoadJSONToMap(jsonPath)
	if err != nil {
		return "Fail To Load Json", 400, err
	}
	// data
	// pureData := rawData
	removeFalsyValues(rawData)
	arrayIgnore := strings.Split(globalData["fieldIgnore"], ",")

	params, ok := rawData["params"].(map[string]interface{})
	if !ok {
		return "No Params In JSON", 400, nil
	}
	for i := 0; i < len(arrayIgnore); i++ {
		params[arrayIgnore[i]] = ""
	}

	paramsPure, ok := pureData["params"].(map[string]interface{})
	if !ok {
		return "No Params In JSON", 400, nil
	}
	flagWA := 1
	if globalData["isPhone"] == "1" {
		noWa := checkString(params["x_pic_phone"])
		flagWA, _ = postWACheck(noWa, globalData["waLink"])

	}

	mapSimCard := stringToMap(globalData["simCard"])
	snEDC := fmt.Sprintf("%d", checkInt(params["x_sn_edc_new"]))
	simCard := fmt.Sprintf("%d", checkInt(params["x_simcard_new"]))
	if mapSimCard[snEDC] == 1 || mapSimCard[simCard] == 1 {
		return "Tolong Isi SN EDC / Simcard EDC yang sesuai", 400, nil
	}

	coo, err := PostLoginOdoo(globalData["linkLogin"], login, pass)
	if err != nil {
		return "Login Odoo Fail", 500, err
	}
	reasonCode := checkInt(params["x_reason_code_id"])
	companyID := checkInt(params["company_id"])
	partnerId := checkInt(params["partner_id"])
	if globalData["isDIstance"] == "1" && companyID == 15 {
		srcLong := checkString(params["x_longtitude"])
		srcLat := checkString(params["x_latitude"])
		dstLat, dstLong, err := getDataPosition(db, partnerId)
		if err != nil {
			fmt.Println("err check db long lat", err)
		} else {
			flagDistanceAI = checkDistanceFull(srcLong, srcLat, dstLat, dstLong)
		}

	}
	supplyThermal := checkInt(params["x_supply_thermal"])
	maxThermal, err := strconv.Atoi(globalData["maxThermal"])
	if err != nil {
		return "Config Max Thermal Err", 500, err
	}
	if supplyThermal > maxThermal {
		return "Thermal Error", 400, err

	}
	isFotoSafe, err := checkPhoto(globalData["CheckPicture"], idTask, globalData["PATH_FULL"])
	if !isFotoSafe {
		return err.Error(), 400, nil
	}
	checkFullCompany := stringToMap(globalData["companyFull"])
	if checkFullCompany[fmt.Sprintf("%d", companyID)] == 1 {
		x_source := checkString(params["x_source"])
		ceatSheet := checkBool(params["x_ceasheet_fix"])
		isFotoSafeDeep, err := checkPhotoDeep(idTask, globalData["PATH_FULL"], x_source, ceatSheet)
		if !isFotoSafeDeep {
			return err.Error(), 400, nil
		}
	}
	mapReasonCompany := stringToMap(globalData["reasonCompany"])
	// stringWithPhoto, err := insertPhotoJson(rawData, globalData["ArrayPicture"], globalData["PATH_FULL"], idTask)
	// if err != nil {
	// 	return "Error Unify json and photo", 400, err
	// }

	stringWithPhotoAI, err := insertPhotoJson(pureData, globalData["ArrayPicture"], globalData["PATH_FULL"], idTask)
	if err != nil {
		return "Error Unify json and photo", 400, err
	}
	if mapReasonCompany[fmt.Sprintf("%d", reasonCode)] == 1 {
		teknisi := getSecondArray(paramsPure, "technician_id")

		type2 := getSecondArray(paramsPure, "x_ticket_type2")
		spk := getSecondArray(paramsPure, "helpdesk_ticket_id")
		product := getSecondArray(paramsPure, "x_product")
		sn := getSecondArray(paramsPure, "x_studio_edc")
		codeDashboard, resDashboard := postDashboard(string(stringWithPhotoAI), globalData["linkPending"], checkStringTanpa(params["x_cimb_master_tid"]), teknisi, companyID, reasonCode, checkStringTanpa(params["x_no_task"]), checkStringTanpa(params["x_task_type"]), checkStringTanpa(params["x_merchant"]), checkStringTanpa(params["x_keterangan"]), checkStringTanpa(params["x_sla_deadline"]), type2, spk, checkStringTanpa(params["x_received_datetime_spk"]), checkStringTanpa(params["x_title_cimb"]), checkStringTanpa(params["x_cimb_tid2"]), checkStringTanpa(params["x_cimb_master_mid"]), product, sn, checkStringTanpa(params["x_studio_alamat"]), "-")
		err := insertLog(db, idTask, codeDashboard, resDashboard, "Send PENDING")
		if codeDashboard == 200 {
			srcDir := fmt.Sprintf("%s/%s", globalData["PATH_FULL"], idTask)
			dstDir := fmt.Sprintf("%s/%s", globalData["BIN_FULL"], idTask)
			_ = MoveFolder(srcDir, dstDir)
		}
		if err != nil {
			fmt.Println(err)
		}
		return "Pending Ditolak, Hubungi tim Technical Assistant", 400, nil
	}
	if flagWA == 400 {
		return "Invalid Phone Number (WA CHECKING)", 400, nil
	}

	if globalData["isAI"] == "1" {
		mapThermal := stringToMap(globalData["thermal"])
		if mapThermal[fmt.Sprintf("%d", checkInt(params["x_ticket_type2"]))] == 1 {
			if checkInt(params["x_supply_thermal"]) == 0 {
				return "Thermal Wajib Diisi Tolong Cek Ulang Deskripsi", 400, nil
			}
		}

		codeAI, resAI := postFullAi(stringWithPhotoAI, flagDistanceAI, globalData["linkAI"])
		insertLog(db, idTask, codeAI, resAI, "POST AI")
		if codeAI != 200 {
			teknisi := getSecondArray(paramsPure, "technician_id")

			type2 := getSecondArray(paramsPure, "x_ticket_type2")
			spk := getSecondArray(paramsPure, "helpdesk_ticket_id")
			product := getSecondArray(paramsPure, "x_product")
			sn := getSecondArray(paramsPure, "x_studio_edc")
			codeDashboard, resDashboard := postDashboard(string(stringWithPhotoAI), globalData["linkErrorAi"], checkStringTanpa(params["x_cimb_master_tid"]), teknisi, companyID, reasonCode, checkStringTanpa(params["x_no_task"]), checkStringTanpa(params["x_task_type"]), checkStringTanpa(params["x_merchant"]), checkStringTanpa(params["x_keterangan"]), checkStringTanpa(params["x_sla_deadline"]), type2, spk, checkStringTanpa(params["x_received_datetime_spk"]), checkStringTanpa(params["x_title_cimb"]), checkStringTanpa(params["x_cimb_tid2"]), checkStringTanpa(params["x_cimb_master_mid"]), product, sn, checkStringTanpa(params["x_studio_alamat"]), resAI)
			err := insertLog(db, idTask, codeDashboard, resDashboard, "Send Error AI")

			if err != nil {
				fmt.Println(err)
			}
			if codeDashboard == 200 {
				srcDir := fmt.Sprintf("%s/%s", globalData["PATH_FULL"], idTask)
				dstDir := fmt.Sprintf("%s/%s", globalData["BIN_FULL"], idTask)
				_ = MoveFolder(srcDir, dstDir)
			}
			return resAI, codeAI, nil
		}
	}
	removeFalsyValues(rawData)
	modifiedJSON, err := json.Marshal(rawData)
	if err != nil {
		fmt.Println("Error:", err)
		return "Fail data into string", 400, nil
	}

	go postFullOdoo(globalData["linkUpdate"], globalData["linkFileStore"], stringWithPhotoAI, string(modifiedJSON), coo, idTask, db, globalData["PATH_FULL"], globalData["BIN_FULL"])

	return "Success", 200, nil
}

func postFileStore(taskData string, urlFileStore string) (int, string) {

	client := &http.Client{}

	// Create a cookiejar and add the session ID cookie to it
	// fmt.Println(taskData)

	// Send a POST request with the JSON data and cookies
	fmt.Println("linknya", urlFileStore)
	resp, err := client.Post(urlFileStore, "application/json", bytes.NewBuffer([]byte(taskData)))
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err.Error()
	}

	return resp.StatusCode, string(body)
}
func postFullOdoo(taskURL string, urlFileStore string, taskData string, taskString string, sessionIDCookie []*http.Cookie, id string, db *sql.DB, root string, rootBackup string) {
	// Convert taskData struct to JSON
	// payload, err := json.Marshal(taskData)
	// if err != nil {
	// 	return "", err
	// }
	fmt.Println(taskURL)
	// err := os.WriteFile("output.txt", []byte(taskString), 0777)
	// if err != nil {
	// 	panic(err)
	// }
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(taskData), &data); err != nil {
		fmt.Println("Error:", err)
		return
	}
	ch, rett := postFileStore(taskData, urlFileStore)
	_ = insertLog(db, id, ch, rett, "POST FILESTORE")
	if ch != 200 {
		fmt.Println(rett)
		return
	}
	// Remove keys with false, "", or 0 values
	// removeFalsyValues(data)

	// Marshal the modified data back to JSON
	// modifiedJSON, err := json.Marshal(taskString)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }

	// Create a new HTTP client
	client := &http.Client{}
	fmt.Println("fisinisdafsfadsf")
	// Create a cookiejar and add the session ID cookie to it
	jar, _ := cookiejar.New(nil)
	urlParsed, err := url.Parse(taskURL)
	if err != nil {
		fmt.Println("1")
		fmt.Println(err)
		return
	}
	jar.SetCookies(urlParsed, sessionIDCookie)
	client.Jar = jar

	// Send a POST request with the JSON data and cookies
	resp, err := client.Post(taskURL, "application/json", bytes.NewBuffer([]byte(taskString)))
	if err != nil {
		fmt.Println("2")
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("3")
		fmt.Println(err)
		return
	}
	_ = insertLog(db, id, resp.StatusCode, string(body), "POST ODOO")
	var respObject Response
	err55 := json.Unmarshal(body, &respObject)
	if err55 != nil {
		fmt.Println("4")
		fmt.Println(err55)

		return
	}

	if respObject.Result.Message == "Success" {
		srcDir := fmt.Sprintf("%s/%s", root, id)
		dstDir := fmt.Sprintf("%s/%s", rootBackup, id)
		_ = MoveFolder(srcDir, dstDir)

		fmt.Println("bro")
	} else {

	}

	return
}
func postWACheck(data string, urlWA string) (int, error) {
	params := url.Values{}
	params.Add("phone", data)
	fullURL := fmt.Sprintf("%s?%s", urlWA, params.Encode())
	resp, err := http.Get(fullURL)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return 0, err
	}

	fmt.Println("Response:", string(body))
	return resp.StatusCode, nil
}

func postDashboard(taskData string, url string, tid string, teknisi string, com int, res int, wo string, tipe string, merchant string, keterangan string, sla string, tik2 string, spk string, receive string, tittle string, tid2 string, mid string, productX string, sn string, alamat string, reason string) (int, string) {

	client := &http.Client{}

	// Create a cookiejar and add the session ID cookie to it

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(taskData)))
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("tid", tid)
	req.Header.Set("tech", teknisi)
	req.Header.Set("com", fmt.Sprintf("%d", com))
	req.Header.Set("res", fmt.Sprintf("%d", res))
	req.Header.Set("wo", wo)
	req.Header.Set("tip", tipe)
	req.Header.Set("mer", merchant)
	req.Header.Set("ket", sanitizeIt(keterangan))
	req.Header.Set("sla", sla)
	req.Header.Set("tik", tik2)
	req.Header.Set("spk", spk)
	req.Header.Set("rcv", receive)
	req.Header.Set("tit", sanitizeIt(tittle))

	req.Header.Set("tid2", tid2)
	req.Header.Set("mid", mid)
	req.Header.Set("edc", productX)
	req.Header.Set("sn", sn)
	req.Header.Set("alamat", alamat)
	req.Header.Set("als", sanitizeIt(reason))
	// req.Header.Set("tid", tid)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err.Error()
	}

	return resp.StatusCode, string(body)
}

func postFullAi(data string, flag int, urlAi string) (int, string) {
	// Convert taskData struct to JSON
	// payload, err := json.Marshal(taskData)
	// if err != nil {
	// 	return "", err
	// }

	// Remove keys with false, "", or 0 values
	// removeFalsyValues(data)

	// // Marshal the modified data back to JSON
	// modifiedJSON, err := json.Marshal(data)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return 0, err.Error()
	// }

	// Create a new HTTP client
	client := &http.Client{}

	// Create a cookiejar and add the session ID cookie to it

	// Send a POST request with the JSON data and cookies
	req, err := http.NewRequest("POST", urlAi, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return 0, err.Error()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Distance", fmt.Sprintf("%d", flag))

	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err.Error()
	}

	return resp.StatusCode, string(body)
}
