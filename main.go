package main

// gọi thư viện
import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "modernc.org/sqlite"
)

//go:embed templates
var templateFiles embed.FS

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type Post struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	CategoryID int    `json:"category_id"`
}
type User struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
}

var db *sql.DB

// Hàm băm mật khẩu cơ bản bằng SHA256
func hashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

// Hàm khởi tạo Database
func initDB() {
	var err error
	_, errCheck := os.Stat("database.db")
	dbExists := !os.IsNotExist(errCheck)

	db, err = sql.Open("sqlite", "database.db")
	if err != nil {
		log.Fatal(err)
	}

	if !dbExists {
		fmt.Println("Phát hiện lần chạy đầu tiên. Đang khởi tạo Database từ web.sql...")
		sqlBytes, err := os.ReadFile("web.sql")
		if err != nil {
			log.Fatal("Lỗi: Không tìm thấy file web.sql! ", err)
		}
		_, err = db.Exec(string(sqlBytes))
		if err != nil {
			log.Fatal("Lỗi khi chạy lệnh SQL: ", err)
		}

		// Tạo sẵn một tài khoản Admin mặc định để test đăng bài
		adminPass := hashPassword("123456")
		db.Exec("INSERT INTO users (full_name, email, password_hash, role) VALUES (?, ?, ?, ?)", "Quản Trị Viên", "admin@example.com", adminPass, "admin")

		fmt.Println("Đã nạp dữ liệu từ web.sql và tạo Admin thành công!")
	} else {
		fmt.Println("Đã kết nối với Database hiện tại thành công!")
	}
}

// Hàm main
func main() {
	initDB()

	// 1. API Đăng nhập
	http.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var creds User
			json.NewDecoder(r.Body).Decode(&creds)
			hashed := hashPassword(creds.Password)

			var user User
			err := db.QueryRow("SELECT id, full_name, email, role FROM users WHERE email = ? AND password_hash = ?", creds.Email, hashed).Scan(&user.ID, &user.FullName, &user.Email, &user.Role)
			if err != nil {
				http.Error(w, `{"error": "Sai email hoặc mật khẩu"}`, http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(user)
		}
	})

	// 2. API Đăng ký
	http.HandleFunc("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var newUser User
			json.NewDecoder(r.Body).Decode(&newUser)
			hashed := hashPassword(newUser.Password)

			_, err := db.Exec("INSERT INTO users (full_name, email, password_hash, role) VALUES (?, ?, ?, ?)", newUser.FullName, newUser.Email, hashed, "student")
			if err != nil {
				// Tạm thời in thẳng biến err ra ngoài để xem lỗi thực sự là gì
				http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"message": "Đăng ký thành công!"})
		}
	})

	// 3. API Danh mục
	http.HandleFunc("/api/categories", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			rows, _ := db.Query("SELECT category_id, category_name FROM categories")
			defer rows.Close()
			var cats []Category
			for rows.Next() {
				var c Category
				rows.Scan(&c.ID, &c.Name)
				cats = append(cats, c)
			}
			json.NewEncoder(w).Encode(cats)
		}
	})

	// 4. API Bài Viết (CRUD)
	http.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			idQuery := r.URL.Query().Get("id")
			if idQuery != "" {
				row := db.QueryRow("SELECT id, title, content, category_id FROM posts WHERE id = ?", idQuery)
				var p Post
				if err := row.Scan(&p.ID, &p.Title, &p.Content, &p.CategoryID); err != nil {
					http.Error(w, `{"error": "Không tìm thấy bài viết"}`, http.StatusNotFound)
					return
				}
				json.NewEncoder(w).Encode(p)
				return
			}

			categoryQuery := r.URL.Query().Get("category_id")
			var rows *sql.Rows
			if categoryQuery != "" {
				catID, _ := strconv.Atoi(categoryQuery)
				rows, _ = db.Query("SELECT id, title, content, category_id FROM posts WHERE category_id = ? ORDER BY id DESC", catID)
			} else {
				rows, _ = db.Query("SELECT id, title, content, category_id FROM posts ORDER BY id DESC")
			}
			defer rows.Close()

			var posts []Post
			for rows.Next() {
				var p Post
				rows.Scan(&p.ID, &p.Title, &p.Content, &p.CategoryID)
				posts = append(posts, p)
			}
			if posts == nil {
				posts = []Post{}
			}
			json.NewEncoder(w).Encode(posts)
			return
		}

		if r.Method == http.MethodPost {
			var newPost Post
			json.NewDecoder(r.Body).Decode(&newPost)
			_, err := db.Exec("INSERT INTO posts (title, content, category_id) VALUES (?, ?, ?)", newPost.Title, newPost.Content, newPost.CategoryID)
			if err != nil {
				http.Error(w, `{"error": "Lỗi lưu dữ liệu"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"message": "Đăng bài thành công!"})
			return
		}
	})

	frontend, _ := fs.Sub(templateFiles, "templates")
	http.Handle("/", http.FileServer(http.FS(frontend)))
	fmt.Println("Giao diện đã sẵn sàng tại http://localhost:8080/student_handbook.html")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
