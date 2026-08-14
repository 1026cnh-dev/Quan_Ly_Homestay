-- Tạo cơ sở dữ liệu
CREATE DATABASE IF NOT EXISTS mocyan_homestay CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE mocyan_homestay;

-- 1. Bảng quản lý Nhân viên/Sale (Mở rộng thêm so với UI)
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    role ENUM('admin', 'sale') DEFAULT 'sale',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Bảng quản lý Căn/Phòng (Homestay Rooms)
CREATE TABLE rooms (
    id INT AUTO_INCREMENT PRIMARY KEY,
    room_key VARCHAR(50) NOT NULL UNIQUE, -- VD: 'mocYen', 'soc', 'mocLam'
    name VARCHAR(100) NOT NULL,
    capacity_desc VARCHAR(100) NOT NULL, -- VD: '4 - max 12 pax'
    base_price INT NOT NULL, -- Lưu số nguyên (VD: 2000000)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 3. Bảng quản lý Hình ảnh của Căn (Room Images)
-- Tách riêng vì 1 căn có nhiều ảnh (Quan hệ 1-N)
CREATE TABLE room_images (
    id INT AUTO_INCREMENT PRIMARY KEY,
    room_id INT NOT NULL,
    image_url VARCHAR(255) NOT NULL, -- Đường dẫn ảnh (VD: 'anh/MocYen/1.png')
    is_fallback BOOLEAN DEFAULT FALSE, -- Đánh dấu nếu là ảnh dự phòng (Unsplash)
    display_order INT DEFAULT 0, -- Thứ tự sắp xếp ảnh
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

-- 4. Bảng lưu Trạng thái đặt phòng (Booking States)
-- Thay thế cho LocalStorage key: YYYY_MM_DD_roomKey
CREATE TABLE bookings (
    id INT AUTO_INCREMENT PRIMARY KEY,
    room_id INT NOT NULL,
    booking_date DATE NOT NULL, -- Lưu theo chuẩn YYYY-MM-DD
    status ENUM('booked', 'reserved') NOT NULL, -- Đỏ hoặc Vàng
    sale_id INT, -- Theo dõi Sale nào đã giữ/đặt chỗ (khóa ngoại nối với users)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (sale_id) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE KEY unique_room_date (room_id, booking_date) -- Ràng buộc: 1 phòng 1 ngày chỉ có 1 trạng thái
);
-- Thêm các căn (Rooms)
INSERT INTO rooms (id, room_key, name, capacity_desc, base_price) VALUES
(1, 'mocYen', 'Mộc Yên', '4 - max 12 pax', 2000000),
(2, 'soc', 'Sóc', '4 - max 8 pax', 1500000),
(3, 'mocLam', 'Mộc Lam', '4 - max 8 pax', 1200000);

-- Thêm hình ảnh chính (Main Images)
INSERT INTO room_images (room_id, image_url, is_fallback, display_order) VALUES
(1, 'anh/MocYen/1.png', FALSE, 1),
(1, 'anh/MocYen/2.png', FALSE, 2),
(1, 'anh/MocYen/3.png', FALSE, 3),
(2, 'anh/Soc/1.png', FALSE, 1),
(2, 'anh/Soc/2.png', FALSE, 2),
(2, 'anh/Soc/3.png', FALSE, 3),
(3, 'anh/MocLam/1.png', FALSE, 1),
(3, 'anh/MocLam/2.png', FALSE, 2);

-- Thêm hình ảnh dự phòng (Fallback Images)
INSERT INTO room_images (room_id, image_url, is_fallback, display_order) VALUES
(1, 'https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&w=1200&q=80', TRUE, 4),
(2, 'https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=1200&q=80', TRUE, 4),
(3, 'https://images.unsplash.com/photo-1513694203232-719a280e022f?auto=format&fit=crop&w=1200&q=80', TRUE, 4);

-- Thử thêm một vài trạng thái đặt phòng mẫu (Bookings)
-- Ngày 14/08/2026: Mộc Yên đã bán, Sóc giữ chỗ
INSERT INTO bookings (room_id, booking_date, status) VALUES
(1, '2026-08-14', 'booked'),
(2, '2026-08-14', 'reserved');
