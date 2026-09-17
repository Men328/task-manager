# errorcode — bộ mã lỗi chuẩn

Nguồn sự thật duy nhất cho mã lỗi của toàn hệ thống:
**`error_codes.json`** (Go embed, frontend dùng bản mirror).

## Vì sao

Backend trước đây trả message kỹ thuật (HTTP status, URL, text nội bộ) và frontend show thẳng lên
notification — vừa xấu vừa khó dịch. Nay mọi lỗi đi kèm **mã lỗi chuẩn**; FE map mã → message đẹp
theo ngôn ngữ, mã lạ thì fallback theo HTTP status.

## Định dạng

```json
{
  "version": "1.0.0",
  "domain": "taskmanager",
  "defaultCode": "COMMON_INTERNAL",
  "codes": {
    "TASK_TITLE_REQUIRED": {
      "owner": "task",
      "grpcCode": "InvalidArgument",
      "httpStatus": 400,
      "message": { "vi": "Tiêu đề công việc là bắt buộc.", "en": "Task title is required." }
    }
  }
}
```

- `owner`: `common` | `identity` | `task` | `client` (mã phía FE, không có `grpcCode`).
- `grpcCode`: tên `codes.Code` để suy ra gRPC status (và HTTP status qua grpc-gateway).
- `httpStatus`: chỉ để tra cứu/đối chiếu (grpc-gateway tự map từ `grpcCode`).
- `message`: text hiển thị cho người dùng, theo ngôn ngữ.

## Backend dùng

```go
import "taskmanager/common/errorcode"

return errorcode.Error(errorcode.TaskTitleRequired, "title là bắt buộc")
```

`errorcode.Error` tạo gRPC status kèm `google.rpc.ErrorInfo` (`reason` = mã lỗi, `domain` =
`taskmanager`). grpc-gateway render detail này vào `details[]` của JSON lỗi:

```json
{
  "code": 3,
  "message": "title là bắt buộc",
  "details": [
    { "@type": "type.googleapis.com/google.rpc.ErrorInfo", "reason": "TASK_TITLE_REQUIRED", "domain": "taskmanager" }
  ]
}
```

> Gateway phải import `google.golang.org/genproto/googleapis/rpc/errdetails` (blank import trong
> `cmd/http/server.go`) để protojson resolve được `Any` khi marshal. Thiếu import này sẽ ra
> `{"code":13,"message":"failed to marshal error message"}`.

## Thêm / sửa mã lỗi

1. Sửa `common/errorcode/error_codes.json` (thêm entry + message vi/en).
2. Thêm hằng số tương ứng trong `errorcode.go` (nếu backend dùng) — test sẽ fail nếu thiếu.
3. `make error-codes` để sinh mirror cho FE (`frontend/src/config/error_codes.json`).
4. `make test` (Go) + `make check-error-codes`.

`errorcode_test.go` giữ cho JSON và hằng số Go không lệch nhau, đồng thời kiểm tra `ErrorInfo`
được marshal ra JSON đúng.
