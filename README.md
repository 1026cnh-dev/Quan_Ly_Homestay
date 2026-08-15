# Hệ thống Quản lý Đặt phòng Homestay

Hệ thống quản lý đặt phòng, theo dõi trạng thái giữ chỗ và kho hình ảnh dành cho bộ phận Sale của Homestay Đà Lạt. Dự án được thiết kế nhẹ, tốc độ phản hồi nhanh và dễ dàng triển khai.

## ✨ Tính năng nổi bật

- Theo dõi lịch đặt phòng: Giao diện trực quan xem trạng thái phòng (Trống, Giữ chỗ, Đã có khách) theo từng ngày trong tháng.
- **Quản lý linh hoạt:** Dễ dàng thao tác click để thay đổi trạng thái phòng trực tiếp trên bảng.
- **Cấu hình giá:** Hỗ trợ cài đặt giá cơ bản cho từng hạng phòng và tùy chỉnh giá riêng cho các ngày lễ, cuối tuần.
- **Thư viện hình ảnh:** Tích hợp bộ sưu tập ảnh cho từng phòng (Mộc Yên, Sóc, Mộc Lam) với trình xem ảnh (Lightbox) có tính năng tải xuống trực tiếp.
- **Giao diện thân thiện:** Responsive hoàn toàn, hiển thị tốt trên cả điện thoại di động và máy tính nhờ Tailwind CSS.

## 🛠 Công nghệ sử dụng

- **Frontend:** HTML5, Tailwind CSS (via CDN), Vanilla JavaScript.
- **Backend:** Go (Golang) sử dụng thư viện chuẩn `net/http`.
- **Database:** SQLite (thông qua driver `modernc.org/sqlite`).
