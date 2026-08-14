package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq" // Thư viện PostgreSQL
)

var db *sql.DB

type BookingRequest struct {
	RoomKey string `json:"roomKey"`
	Date    string `json:"date"`
	Status  string `json:"status"`
}

func initDB() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("❌ LỖI: Chưa có đường dẫn DATABASE_URL")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Lỗi kết nối Supabase:", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS rooms (
		id SERIAL PRIMARY KEY,
		room_key VARCHAR(50) UNIQUE,
		name VARCHAR(100),
		capacity_desc VARCHAR(100),
		base_price INTEGER
	);`)
	if err != nil {
		log.Fatal("Lỗi tạo bảng rooms:", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS bookings (
		id SERIAL PRIMARY KEY,
		room_id INTEGER REFERENCES rooms(id),
		booking_date VARCHAR(20),
		status VARCHAR(20),
		UNIQUE(room_id, booking_date)
	);`)
	if err != nil {
		log.Fatal("Lỗi tạo bảng bookings:", err)
	}

	db.Exec(`INSERT INTO rooms (room_key, name, capacity_desc, base_price) VALUES
		('mocYen', 'Mộc Yên', '4 - max 12 pax', 2000000),
		('soc', 'Sóc', '4 - max 8 pax', 1500000),
		('mocLam', 'Mộc Lam', '4 - max 8 pax', 1200000)
		ON CONFLICT (room_key) DO NOTHING;`)
}

func main() {
	initDB()
	defer db.Close()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/bookings", bookingsHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("🚀 SERVER ĐANG CHẠY TẠI CỔNG:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func bookingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		year := r.URL.Query().Get("year")
		month := r.URL.Query().Get("month")

		// SỬA LỖI Ở ĐÂY: Dùng biến sqlMonth riêng để truy vấn, giữ nguyên biến month gốc
		sqlMonth := month
		if len(sqlMonth) == 1 {
			sqlMonth = "0" + sqlMonth
		}

		query := `
			SELECT r.room_key, CAST(SUBSTRING(b.booking_date FROM 9 FOR 2) AS INTEGER), b.status
			FROM bookings b JOIN rooms r ON b.room_id = r.id
			WHERE b.booking_date LIKE $1
		`
		datePattern := fmt.Sprintf("%s-%s-%%", year, sqlMonth)
		rows, err := db.Query(query, datePattern)
		if err != nil {
			http.Error(w, `{"error": "Lỗi truy vấn DB"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		bookingState := make(map[string]string)
		for rows.Next() {
			var roomKey, status string
			var day int
			if err := rows.Scan(&roomKey, &day, &status); err == nil {
				// Trả về số "8" thay vì "08" để khớp 100% với Frontend
				key := fmt.Sprintf("%s_%s_%d_%s", year, month, day, roomKey)
				bookingState[key] = status
			}
		}
		json.NewEncoder(w).Encode(bookingState)

	case "POST":
		var req BookingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Dữ liệu không hợp lệ"}`, http.StatusBadRequest)
			return
		}

		var roomID int
		err := db.QueryRow("SELECT id FROM rooms WHERE room_key = $1", req.RoomKey).Scan(&roomID)
		if err != nil {
			http.Error(w, `{"error": "Không tìm thấy phòng"}`, http.StatusInternalServerError)
			return
		}

		if req.Status == "" {
			db.Exec("DELETE FROM bookings WHERE room_id = $1 AND booking_date = $2", roomID, req.Date)
		} else {
			query := `
				INSERT INTO bookings (room_id, booking_date, status) VALUES ($1, $2, $3)
				ON CONFLICT(room_id, booking_date) DO UPDATE SET status = EXCLUDED.status;
			`
			_, err := db.Exec(query, roomID, req.Date, req.Status)
			if err != nil {
				fmt.Println("Lỗi lưu DB:", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
