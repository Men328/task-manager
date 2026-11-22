# Flow simulator (docs/system/simulator) — HTML/CSS/JS tĩnh, không cần build.
# Nội dung trang đến từ build context phụ `site` (khai báo trong docker-compose.yml),
# vì .dockerignore ở gốc repo loại thư mục docs/.
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=site . /usr/share/nginx/html

EXPOSE 5500
