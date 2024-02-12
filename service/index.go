package service

import (
	"context"
	"database/sql"
	"echolabstack/model"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron"

	getfilelist "github.com/tanaikech/go-getfilelist"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type FileInfo struct {
	Name        string
	ID          string
	CreatedTime string
}



func NewAPI(c echo.Context) error {
	return c.String(http.StatusOK, "New API created")
}

func PdfAPI(c echo.Context) error {
	pdfPath := "static/Echo_static.pdf"
	return c.File(pdfPath)

}

var db *sql.DB
var err error
var lastBatchTime time.Time
var count int16 = 0

func Database() {
	db, err = sql.Open("sqlite3", "./subscribers.db")
	if err != nil {
		panic(err)
	}
	// defer db.Close()

	// Create table if not exists
	createTable := `
	CREATE TABLE IF NOT EXISTS subscribers (
  		email TEXT PRIMARY KEY,
   		posted BOOLEAN,
		batch_id INTEGER DEFAULT 0,
		FOREIGN KEY (batch_id) REFERENCES batch(batch_id)
	);  
	`
	_, err = db.Exec(createTable)

	createTable1 := `
	CREATE TABLE IF NOT EXISTS batch (
		batch_id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(createTable1)

	driveTable := `
    CREATE TABLE IF NOT EXISTS googledrive (
        filename TEXT PRIMARY KEY
    );
    `
	_, err = db.Exec(driveTable)
}
func EmailIDAPI(c echo.Context) error {
	// Parse request body
	var email model.Email
	if err := c.Bind(&email); err != nil {
		return err
	}
	count = 0
	// Insert email into database
	_, err := db.Exec("INSERT INTO subscribers (email,posted) VALUES (?, ?)", email.Email, email.Posted)

	if err != nil {
		fmt.Println("Error inserting into database:", err)
		return c.String(http.StatusConflict, "Email already subscribed")
	}

	StartCron()
	return c.String(http.StatusCreated, "Subscribed successfully")
}
func StartCron() {
	c := cron.New()
	_ = c.AddFunc("1 * * ? * *", post)
	c.Start()
}

// @cron(run every 1 min)
func post() {

	// Create a new batch
	var newBatchID int64
	// if count == 0{
		newBatchID, err = insertBatch()
		if err != nil {
			log.Println("Error creating batch:", err)
			return
		}
		log.Printf("New batch created: %d", newBatchID)
		if err != nil {
			log.Fatal(err)
		}
	// }
	
	_, err = db.Exec("UPDATE subscribers SET batch_id = ? WHERE posted = false", newBatchID)
	stmt, err := db.Prepare("SELECT email FROM subscribers WHERE posted = false AND batch_id = ?")
	if err != nil {
		log.Println("error*****")
		panic(err)
	}
	mail, err := stmt.Query(newBatchID)
	if err != nil {
		log.Println("error")
		panic(err)
	}
	defer mail.Close()
	var emailIds []string

	// Iterate through the mail
	for mail.Next() {
		var email string
		if err := mail.Scan(&email); err != nil {
			log.Fatal(err)
		}
		emailIds = append(emailIds, email)
	}

	if len(emailIds) > 0 {
		sendMail(emailIds)
	}

	_, err = db.Exec("UPDATE subscribers SET posted = true WHERE posted = false AND batch_id = ?", newBatchID)

	if err != nil {
		log.Fatal(err)
	}
	// count ++
}

func sendMail(whoSubscribed []string) {
	//send email(admin@netxd.com,whoSubscribed)
	log.Println(whoSubscribed)
}

func insertBatch() (int64, error) {
	result, err := db.Exec("INSERT INTO batch DEFAULT VALUES")
	if err != nil {
		return -1, err
	}
	batchID, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return batchID, nil
}

//recapcha

func CapcheAPI(c echo.Context) error {

	token := c.FormValue("token")
	recaptchaResponse, err := verifyRecaptchaToken(token)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error verifying reCAPTCHA token")
		return err
	}

	if recaptchaResponse.Success && recaptchaResponse.Score >= 0.5 {
		fmt.Println("Recapcha score", recaptchaResponse.Score)
		return c.String(http.StatusOK, fmt.Sprintf("reCAPTCHA token successfully verified with score %f!", recaptchaResponse.Score))
	} else {
		fmt.Println("Recapcha score", recaptchaResponse.Score)

		return c.String(http.StatusBadRequest, "reCAPTCHA token verification failed or score < 0.5")
	}
}
func verifyRecaptchaToken(token string) (*model.RecaptchaResponse, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://www.google.com/recaptcha/api/siteverify", nil)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Add("secret", "6Le6wG8pAAAAAB56Y6W80WqUMxDsm5DVVf4MpJRe")
	query.Add("response", token)
	req.URL.RawQuery = query.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var recaptchaResponse model.RecaptchaResponse
	err = json.NewDecoder(resp.Body).Decode(&recaptchaResponse)
	if err != nil {
		return nil, err
	}

	return &recaptchaResponse, nil
}
func Filehandler(c echo.Context) error {
	folderPath := "/home/thashmigaa/Downloads"

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}

	var pdfFiles []os.FileInfo
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".pdf" {
			pdfFiles = append(pdfFiles, file)
		}
	}

	sort.Slice(pdfFiles, func(i, j int) bool {
		return pdfFiles[i].ModTime().After(pdfFiles[j].ModTime())
	})

	latestPDF := pdfFiles[0]

	file, err := os.Open(filepath.Join(folderPath, latestPDF.Name()))
	if err != nil {
		return err
	}
	defer file.Close()

	// c.Response().Header().Set("Content-Disposition", "attachment; filename=abc.pdf")
	//c.Response().Header().Set("Content-Disposition", "attachment; filename="+latestPDF.Name())

	c.Response().Header().Set("Content-Type", "application/pdf")

	c.Response().Header().Set("Content-Disposition", "inline; filename="+latestPDF.Name())

	if _, err := io.Copy(c.Response().Writer, file); err != nil {
		return err
	}
	return nil
}
func Drive(c echo.Context) error {
	folderID := "13YzbZX8cKDFIIMehjTGZrGfH0q5EcXSr"
	credentialFile := "/home/thashmigaa/Downloads/credential.json"

	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsFile(credentialFile))
	if err != nil {
		log.Fatal(err)
	}

	// When you want to retrieve the file list in the folder,
	res, err := getfilelist.Folder(folderID).Fields("files(name,id,createdTime)").Do(srv)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create a slice to hold file information
	var files []FileInfo

	for _, fileList := range res.FileList {
		for _, file := range fileList.Files {
			fileInfo := FileInfo{
				Name:        file.Name,
				ID:          file.Id,
				CreatedTime: file.CreatedTime,
			}
			files = append(files, fileInfo)
		}
	}

	// Sort files by ModifiedTime

	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedTime > files[j].CreatedTime
	})

	// count := 0
	for _, file := range files {
		// if count == 3 {
		//  break
		// }
		fmt.Println(file.Name, file.CreatedTime)

		_, err := db.Exec("INSERT INTO googledrive (filename) VALUES (?)", file.Name)

		if err != nil {
			fmt.Println("Error inserting into database:", err)
			return c.String(http.StatusConflict, "file already exists")
		} else {

			name := file.Name
			filename, err := os.Create(("/home/thashmigaa/Downloads/" + name))
			if err != nil {
				log.Fatalf("Unable to create file: %v", err)
			}
			defer filename.Close()
			resp, err := srv.Files.Get(file.ID).Download()
			if err != nil {
				log.Fatalf("Unable to download file: %v", err)
			}
			defer resp.Body.Close()
			_, err = io.Copy(filename, resp.Body)
			if err != nil {
				log.Fatalf("Unable to copy file content: %v", err)
			} else {
				log.Println(file.ID)
			}
			return c.String(http.StatusCreated, "files uploaded successfully")
		}

		// count++
	}
	return nil

}
