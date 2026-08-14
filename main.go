package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

type BookingRequest struct {
	RoomKey string `json:"roomKey"`
	Date    string `json:"date"`
	Status  string `json:"status"`
}

type PriceRequest struct {
	Type    string `json:"type"` // "base", "special", "delete_special"
	RoomKey string `json:"roomKey"`
	Date    string `json:"date"`
	Price   int    `json:"price"`
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

	// 1. Bảng phòng (Lưu giá cơ bản)
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

	// 2. Bảng đặt phòng
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

	// 3. Bảng giá ngày lễ/ngày đặc biệt
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS special_prices (
		id SERIAL PRIMARY KEY,
		room_key VARCHAR(50),
		date_str VARCHAR(20),
		price INTEGER,
		UNIQUE(room_key, date_str)
	);`)
	if err != nil {
		log.Fatal("Lỗi tạo bảng special_prices:", err)
	}

	db.Exec(`INSERT INTO rooms (room_key, name, capacity_desc, base_price) VALUES
		('mocYen', 'Mộc Yên', '4 - max 12 pax', 2000000),
		('soc', 'Sóc', '4 - max 8 pax', 1500000),
		('mocLam', 'Mộc Lam', '4 - max 8 pax', 1500000)
		ON CONFLICT (room_key) DO NOTHING;`)
}

func main() {
	initDB()
	defer db.Close()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/bookings", bookingsHandler)
	http.HandleFunc("/api/prices", pricesHandler) // API mới quản lý giá

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
			db.Exec(query, roomID, req.Date, req.Status)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}
}

// Hàm xử lý Giá phòng (MỚI)
func pricesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// Lấy giá cơ bản
		basePrices := make(map[string]int)
		rows, _ := db.Query("SELECT room_key, base_price FROM rooms")
		for rows.Next() {
			var rk string
			var bp int
			rows.Scan(&rk, &bp)
			basePrices[rk] = bp
		}
		rows.Close()

		// Lấy giá ngày lễ trong tháng
		year := r.URL.Query().Get("year")
		month := r.URL.Query().Get("month")
		sqlMonth := month
		if len(sqlMonth) == 1 {
			sqlMonth = "0" + sqlMonth
		}
		specialPrices := make(map[string]int)
		pattern := fmt.Sprintf("%s-%s-%%", year, sqlMonth)
		rows2, _ := db.Query("SELECT room_key, CAST(SUBSTRING(date_str FROM 9 FOR 2) AS INTEGER), price FROM special_prices WHERE date_str LIKE $1", pattern)
		for rows2.Next() {
			var rk string
			var d, p int
			rows2.Scan(&rk, &d, &p)
			key := fmt.Sprintf("%s_%s_%d_%s", year, month, d, rk)
			specialPrices[key] = p
		}
		rows2.Close()

		response := map[string]interface{}{
			"basePrices":    basePrices,
			"specialPrices": specialPrices,
		}
		json.NewEncoder(w).Encode(response)

	case "POST":
		var req PriceRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Type == "base" {
			db.Exec("UPDATE rooms SET base_price = $1 WHERE room_key = $2", req.Price, req.RoomKey)
		} else if req.Type == "special" {
			db.Exec("INSERT INTO special_prices (room_key, date_str, price) VALUES ($1, $2, $3) ON CONFLICT (room_key, date_str) DO UPDATE SET price = EXCLUDED.price", req.RoomKey, req.Date, req.Price)
		} else if req.Type == "delete_special" {
			db.Exec("DELETE FROM special_prices WHERE room_key = $1 AND date_str = $2", req.RoomKey, req.Date)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}
}
