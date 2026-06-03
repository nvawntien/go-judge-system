# API Contract

## Purpose

File này là source of truth cho frontend khi tích hợp với backend Go REST API.

Không tự ý sửa file này nếu backend chưa thay đổi.

Mục tiêu của file:

- Ghi lại contract thực tế đang có trong backend Go.
- Giúp frontend engineer và Codex tích hợp API nhất quán.
- Làm rõ endpoint nào public, endpoint nào cần auth, và endpoint nào chưa chắc chắn.

## Frontend usage rules

- Frontend phải gọi API qua gateway `:8080`.
- Không gọi trực tiếp service ports.
- Không tự bịa thêm endpoint, method, request body, response shape, hoặc auth behavior.
- Nếu backend chưa hoàn thiện một endpoint, chỉ được mock tạm sau khi endpoint đó đã được ghi rõ trong `Unknown / needs confirmation`.
- Mock phải giữ đúng response wrapper của backend.
- Không hardcode mock trực tiếp trong UI component lớn.
- Không tự âm thầm tạo “contract mới” khác với file này.
- Với endpoint protected, frontend phải gửi cookie bằng `credentials: "include"`.
- Không dùng Bearer token.
- Không dùng `localStorage` hoặc `sessionStorage` để lưu auth token.

## Gateway rules

- Gateway là entrypoint duy nhất cho frontend.
- Gateway validate JWT từ cookie `access_token`.
- Với endpoint cần auth, backend không tự parse token từ frontend request; backend đọc claims từ các header do gateway inject:
  - `X-User-ID`
  - `X-Role`
  - `X-Username`
  - `X-Token-Iat`
- Backend hiện không có `PATCH` route.

## Response wrapper

Business API hiện dùng wrapper chuẩn:

```json
{
  "status": "success | error",
  "code": 20000,
  "msg": "optional message",
  "data": {}
}
```

### ApiResponse type

```ts
export type ApiResponse<T> = {
  status: "success" | "error"
  code: number
  msg?: string
  data?: T
}
```

### Success response

- Success response có thể có `data`.
- Success response có thể có `msg`.
- Có endpoint trả cả `msg` lẫn `data`.
- Có endpoint trả `data: null`.

Ví dụ:

```json
{
  "status": "success",
  "code": 20100,
  "msg": "registration successful, please check your email to verify your account",
  "data": null
}
```

### Error response

Error response thường có dạng:

```json
{
  "status": "error",
  "code": 40000,
  "msg": "error message"
}
```

### Common business codes

- `20000`: success
- `20100`: created
- `20001`: updated
- `20002`: deleted
- `20003`: retrieved
- `40000`: bad request
- `40001`: invalid URI params
- `40002`: invalid ID
- `40100`: unauthorized
- `40101`: invalid or expired token
- `40102`: token expired
- `40103`: invalid password
- `40300`: forbidden
- `40400`: not found
- `40401`: account not found
- `40900`: conflict
- `42200`: validation failed
- `42900`: rate limit exceeded
- `50000`: internal server error
- `50004`: redis error

## Auth behavior

### Cookie-based auth

- Auth dùng custom JWT HS256.
- Không dùng NextAuth.
- Không thấy Bearer token flow trong backend hiện tại.
- Login set HTTP-only cookies `access_token` và `refresh_token`.
- Refresh đọc `refresh_token` từ cookie, sau đó set lại cả `access_token` và `refresh_token`.
- Logout và logout-all clear cả hai cookie bằng `SetCookie(..., "", -1, ...)`.
- Gateway validate JWT từ cookie `access_token`.
- Logout-all còn invalid token cũ dựa trên `iat`.

Cookie notes hiện tại:

- `HttpOnly`: yes
- `Secure`: currently `false`
- `Path`: `/`
- SameSite: chưa thấy set rõ trong handler

### Frontend requirements

- Với login state và protected API, luôn gửi `credentials: "include"`.
- Không tự giữ access token ở client state như source of truth.
- Không đọc auth token từ `localStorage` hoặc `sessionStorage`.
- Nên coi cookie là auth source of truth.

### Protected request requirements

Với endpoint protected:

- Gateway validate JWT từ cookie `access_token`.
- Backend đọc claims từ:
  - `X-User-ID`
  - `X-Role`
  - `X-Username`
  - `X-Token-Iat`

## Mock API rules

- Chỉ mock endpoint khi backend chưa sẵn sàng hoặc contract còn chưa chắc chắn.
- Nếu mock, phải mock đúng response wrapper:

```json
{
  "status": "success | error",
  "code": 20000,
  "msg": "optional",
  "data": {}
}
```

- Endpoint chưa chắc phải nằm trong `Unknown / needs confirmation`.
- Không hardcode mock trực tiếp trong component UI lớn.
- Không tự tạo API contract mới mà không cập nhật file này.

## Endpoints

### Auth

#### POST /api/v1/auth/register

**Description:**  
Đăng ký tài khoản mới.

**Auth required:** no

**Handler:**  
`RegisterHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/register.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "full_name": "string",
  "username": "string",
  "email": "string",
  "password": "string"
}
```

**Response:**

Code: `20100`

```json
{
  "status": "success",
  "code": 20100,
  "msg": "registration successful, please check your email to verify your account",
  "data": null
}
```

**Error cases:**

- `40000` invalid request payload
- `40900` email already exists
- `40900` username already exists
- `40900` user already exists
- `42900` rate limit exceeded

**Frontend notes:**

- Password rule ở DTO mới chỉ thấy `required`; domain có thể còn check password weakness.

#### POST /api/v1/auth/login

**Description:**  
Đăng nhập bằng `identifier` và `password`.

**Auth required:** no

**Handler:**  
`LoginHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/login.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "identifier": "string",
  "password": "string"
}
```

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "login successful",
  "data": {
    "access_token": "string",
    "access_expire": 3600,
    "refresh_token": "string",
    "refresh_expire": 604800
  }
}
```

**Error cases:**

- `40000` invalid request payload
- `40100` invalid username or password
- `40300` user is not active
- `40101` invalid or expired token

**Frontend notes:**

- Login set cookie `access_token` và `refresh_token`.
- Frontend nên xem cookie là source of truth cho auth state.

#### POST /api/v1/auth/refresh-token

**Description:**  
Refresh session bằng cookie `refresh_token`.

**Auth required:** no

**Handler:**  
`RefreshTokenHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/refresh_token.go`

**Request params:**

- none

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "token refreshed successfully",
  "data": {
    "access_token": "string",
    "access_expire": 3600,
    "refresh_token": "string",
    "refresh_expire": 604800
  }
}
```

**Error cases:**

- `40100` missing refresh token
- `40101` invalid or expired token

**Frontend notes:**

- Phải gửi `credentials: "include"`.
- Endpoint này đọc `refresh_token` từ cookie và set lại cookie mới.

#### POST /api/v1/auth/logout

**Description:**  
Đăng xuất thiết bị hiện tại.

**Auth required:** yes

**Handler:**  
`LogoutHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/logout.go`

**Request params:**

- none

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "logged out successfully",
  "data": null
}
```

**Error cases:**

- `40100` unauthorized: missing identity headers from gateway
- `40100` unauthorized: missing or invalid token iat
- `40100` unauthorized: token has been invalidated

**Frontend notes:**

- Gateway auth required qua cookie `access_token`.
- Endpoint này clear cả `access_token` và `refresh_token`.

#### POST /api/v1/auth/logout-all

**Description:**  
Đăng xuất mọi thiết bị của user hiện tại.

**Auth required:** yes

**Handler:**  
`LogoutAllHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/logout_all.go`

**Request params:**

- none

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "logged out from all devices successfully",
  "data": null
}
```

**Error cases:**

- `40100` unauthorized
- `50004` failed to validate session

**Frontend notes:**

- Endpoint này clear cookie hiện tại.
- Token cũ sẽ tiếp tục bị reject bởi middleware dựa trên `iat`.

#### POST /api/v1/auth/email/verify

**Description:**  
Verify email bằng token.

**Auth required:** no

**Handler:**  
`VerifyEmailHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/verify_email.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "token": "string"
}
```

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "email verified successfully, your account is now active",
  "data": null
}
```

**Error cases:**

- `40000` invalid request payload
- `40101` invalid or expired token
- `40900` user is already active

**Frontend notes:**

- Public endpoint.

#### POST /api/v1/auth/email/resend-verification

**Description:**  
Gửi lại email verify.

**Auth required:** no

**Handler:**  
`ResendVerificationHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/resend_verification.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "email": "string"
}
```

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "If the email is valid, a link has been sent. Please check your email.",
  "data": null
}
```

**Error cases:**

- `40000` invalid request payload
- `42900` rate limit exceeded
- `40300` you are temporarily blocked due to multiple OTP requests

**Frontend notes:**

- Message có tính generic, không xác nhận email có tồn tại hay không.

#### POST /api/v1/auth/password/forgot

**Description:**  
Bắt đầu flow quên mật khẩu.

**Auth required:** no

**Handler:**  
`ForgotPasswordHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/forgot_password.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "email": "string"
}
```

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "If the email is valid, a link has been sent. Please check your email.",
  "data": null
}
```

**Error cases:**

- `40000` invalid request payload
- `42900` rate limit exceeded

**Frontend notes:**

- Message generic, không xác nhận email có tồn tại hay không.

#### POST /api/v1/auth/password/reset

**Description:**  
Reset mật khẩu bằng token.

**Auth required:** no

**Handler:**  
`ResetPasswordHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/reset_password.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "token": "string",
  "new_password": "string",
  "confirm_password": "string"
}
```

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "password reset successfully, you can now log in with your new password",
  "data": null
}
```

**Error cases:**

- `40000` invalid request payload
- `40000` password and confirm password do not match
- `40101` invalid or expired token

**Frontend notes:**

- `confirm_password` phải bằng `new_password`.

#### PUT /api/v1/auth/password/change

**Description:**  
Đổi mật khẩu khi đã đăng nhập.

**Auth required:** yes

**Handler:**  
`ChangePasswordHandler.Handle`  
File: `services/auth/internal/adapter/inbound/http/handler/auth/change_password.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "current_password": "string",
  "new_password": "string",
  "confirm_password": "string"
}
```

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "msg": "password changed successfully, you can now log in with your new password",
  "data": null
}
```

**Error cases:**

- `40000` invalid request payload
- `40000` new password is the same as the current password
- `40103` incorrect current password
- `40100` unauthorized

**Frontend notes:**

- Phải gửi `credentials: "include"`.

### Problems

#### GET /api/v1/problems

**Description:**  
Lấy danh sách public problems đã publish.

**Auth required:** no

**Handler:**  
`ListProblemsHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/list_problem.go`

**Request params:**

- none

**Query params:**

- `page?: number = 1`
- `limit?: number = 20`
- `difficulty?: "EASY" | "MEDIUM" | "HARD"`
- `search?: string`

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "items": [
      {
        "id": 1,
        "slug": "two-sum",
        "title": "Two Sum",
        "description": "string",
        "difficulty": "EASY",
        "examples": [
          {
            "input": "string",
            "output": "string",
            "explanation": "string"
          }
        ],
        "constraints": "string",
        "hints": ["string"],
        "time_limit": 1,
        "memory_limit": 128,
        "author_id": "string",
        "is_hidden": false,
        "created_at": "string"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

**Error cases:**

- `40000` invalid query parameters

**Frontend notes:**

- User-facing route dùng slug cho detail page.

#### GET /api/v1/problems/{slug}

**Description:**  
Lấy public problem detail bằng slug.

**Auth required:** no

**Handler:**  
`GetProblemHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/get_problem.go`

**Request params:**

- `slug: string`

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "id": 1,
    "slug": "string",
    "title": "string",
    "description": "string",
    "difficulty": "EASY",
    "examples": [
      {
        "input": "string",
        "output": "string",
        "explanation": "string"
      }
    ],
    "constraints": "string",
    "hints": ["string"],
    "time_limit": 1,
    "memory_limit": 128,
    "author_id": "string",
    "is_hidden": false,
    "created_at": "string"
  }
}
```

**Error cases:**

- `40001` invalid URI params
- `40000` invalid problem slug
- `40400` Problem not found

**Frontend notes:**

- `ProblemDetailResponse` hiện tại có cùng shape với `ProblemResponse`.

#### GET /api/v1/my/problems

**Description:**  
Lấy danh sách problem do admin hiện tại sở hữu.

**Auth required:** yes

**Handler:**  
`ListProblemsHandler.HandleMy`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/list_problem.go`

**Request params:**

- none

**Query params:**

- `page?: number = 1`
- `limit?: number = 20`
- `difficulty?: "EASY" | "MEDIUM" | "HARD"`
- `search?: string`

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "limit": 20
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40000` invalid query parameters

**Frontend notes:**

- Phải gửi `credentials: "include"`.

#### GET /api/v1/admin/problems

**Description:**  
Lấy danh sách tất cả problem cho admin hoặc super admin.

**Auth required:** yes

**Handler:**  
`ListProblemsHandler.HandleAdmin`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/list_problem.go`

**Request params:**

- none

**Query params:**

- `page?: number = 1`
- `limit?: number = 20`
- `difficulty?: "EASY" | "MEDIUM" | "HARD"`
- `search?: string`

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "limit": 20
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40000` invalid query parameters

**Frontend notes:**

- Endpoint này có trong gateway và có implementation trong service router.

#### GET /api/v1/admin/problems/{id}

**Description:**  
Lấy problem detail cho admin bằng numeric ID.

**Auth required:** yes

**Handler:**  
`GetProblemHandler.HandleAdmin`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/get_problem.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "id": 1,
    "slug": "string",
    "title": "string",
    "description": "string",
    "difficulty": "EASY",
    "examples": [],
    "constraints": "string",
    "hints": [],
    "time_limit": 1,
    "memory_limit": 128,
    "author_id": "string",
    "is_hidden": true,
    "created_at": "string"
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40001` invalid URI params
- `40400` Problem not found
- `40300` forbidden

**Frontend notes:**

- Admin route dùng ID, không dùng slug.

#### POST /api/v1/admin/problems

**Description:**  
Tạo problem mới.

**Auth required:** yes

**Handler:**  
`CreateProblemHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/create_problem.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "title": "string",
  "slug": "string",
  "description": "string",
  "difficulty": "EASY | MEDIUM | HARD",
  "examples": [
    {
      "input": "string",
      "output": "string",
      "explanation": "string"
    }
  ],
  "constraints": "string",
  "hints": ["string"],
  "time_limit": 1,
  "memory_limit": 128
}
```

**Response:**

Code: `20100`

```json
{
  "status": "success",
  "code": 20100,
  "data": {
    "id": 1,
    "slug": "two-sum"
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40000` invalid request payload
- `40900` Problem already exists

**Frontend notes:**

- New problem entity mặc định `is_hidden = true`.

#### PUT /api/v1/admin/problems/{id}

**Description:**  
Update problem theo ID.

**Auth required:** yes

**Handler:**  
`UpdateProblemHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/update_problem.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:**

```json
{
  "title": "string?",
  "slug": "string?",
  "description": "string?",
  "difficulty": "EASY | MEDIUM | HARD ?",
  "examples": [
    {
      "input": "string",
      "output": "string",
      "explanation": "string"
    }
  ],
  "constraints": "string?",
  "hints": ["string"]?,
  "time_limit": 1?,
  "memory_limit": 128?
}
```

**Response:**

Code: `20001`

```json
{
  "status": "success",
  "code": 20001,
  "msg": "problem updated successfully",
  "data": null
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40001` invalid URI params
- `40000` invalid request payload
- `40400` Problem not found

**Frontend notes:**

- Body là partial update; tất cả field đều optional.

#### DELETE /api/v1/admin/problems/{id}

**Description:**  
Xóa problem theo ID.

**Auth required:** yes

**Handler:**  
`DeleteProblemHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/delete_problem.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20002`

```json
{
  "status": "success",
  "code": 20002,
  "msg": "problem deleted successfully",
  "data": null
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40001` invalid URI params
- `40400` Problem not found

**Frontend notes:**

- Admin route.

#### PUT /api/v1/admin/problems/{id}/publish

**Description:**  
Publish hidden problem.

**Auth required:** yes

**Handler:**  
`PublishProblemHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/publish_problem.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20001`

```json
{
  "status": "success",
  "code": 20001,
  "msg": "problem published successfully",
  "data": null
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40001` invalid URI params
- `40400` Problem not found

**Frontend notes:**

- Role hoặc ownership check diễn ra ở use case.

#### PUT /api/v1/admin/problems/{id}/hide

**Description:**  
Hide published problem.

**Auth required:** yes

**Handler:**  
`HideProblemHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/problem/hide_problem.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20001`

```json
{
  "status": "success",
  "code": 20001,
  "msg": "problem hidden successfully",
  "data": null
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40001` invalid URI params
- `40400` Problem not found

**Frontend notes:**

- Admin route.

#### POST /api/v1/admin/problems/{id}/testcases

**Description:**  
Upload file ZIP testcase cho problem.

**Auth required:** yes

**Handler:**  
`UploadTestCaseHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/test_case/upload_testcase.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:**

```json
{
  "multipart/form-data": true,
  "file": "required"
}
```

**Response:**

Code: `20100`

```json
{
  "status": "success",
  "code": 20100,
  "data": {
    "test_count": 10,
    "version": "string"
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40001` invalid URI params
- `40000` invalid form data
- `40000` Invalid test case

**Frontend notes:**

- Gateway config đánh dấu request này là `is_stream: true`.

### Submissions

#### GET /api/v1/submissions

**Description:**  
Lấy danh sách submissions public hoặc global theo filter.

**Auth required:** no

**Handler:**  
`ListSubmissionsHandler.Handle`  
File: `services/submission/internal/adapter/inbound/http/handler/submission/list_submissions.go`

**Request params:**

- none

**Query params:**

- `page?: number = 1`
- `limit?: number = 20`
- `problem_id?: number`
- `user_id?: string`
- `status?: string`
- `language?: string`

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "items": [
      {
        "id": 1,
        "problem_id": 1,
        "problem_name": "two-sum",
        "user_id": "string",
        "username": "string",
        "language": "GO",
        "status": "ACCEPTED",
        "created_at": "string"
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

**Error cases:**

- `40000` invalid query parameters

**Frontend notes:**

- Route này hiện public trong service router.
- Ý định cuối cùng của behavior public này vẫn cần confirm ở mục cuối file.

#### GET /api/v1/problems/id/{id}/submissions

**Description:**  
Lấy danh sách submissions của một problem theo ID.

**Auth required:** no

**Handler:**  
`ListSubmissionsHandler.HandleProblem`  
File: `services/submission/internal/adapter/inbound/http/handler/submission/list_submissions.go`

**Request params:**

- `id: number`

**Query params:**

- `page?: number = 1`
- `limit?: number = 20`
- `status?: string`
- `language?: string`

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "limit": 20
  }
}
```

**Error cases:**

- `40001` invalid URI params
- `40000` invalid query parameters

**Frontend notes:**

- Path thực tế là `/api/v1/problems/id/{id}/submissions`, không phải slug.

#### POST /api/v1/submissions

**Description:**  
Tạo submission mới.

**Auth required:** yes

**Handler:**  
`CreateSubmissionHandler.Handle`  
File: `services/submission/internal/adapter/inbound/http/handler/submission/create_submission.go`

**Request params:**

- none

**Query params:** none

**Request body:**

```json
{
  "problem_id": 1,
  "problem_name": "two-sum",
  "language": "C | CPP | JAVA | PYTHON | GO | JAVASCRIPT",
  "source_code": "string"
}
```

**Response:**

Code: `20100`

```json
{
  "status": "success",
  "code": 20100,
  "data": {
    "id": 1,
    "problem_id": 1,
    "problem_name": "two-sum",
    "user_id": "string",
    "username": "string",
    "language": "GO",
    "status": "PENDING",
    "created_at": "string"
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40000` invalid request payload
- `40000` Invalid language
- `40000` Invalid source code

**Frontend notes:**

- Tên field là `problem_name`, nhưng trong pipeline judge field này được map vào `problem_slug`.
- Frontend nên coi đây là điểm cần confirm, không nên tự suy diễn âm thầm.

#### GET /api/v1/my/submissions

**Description:**  
Lấy danh sách submissions của user hiện tại.

**Auth required:** yes

**Handler:**  
`ListSubmissionsHandler.HandleMy`  
File: `services/submission/internal/adapter/inbound/http/handler/submission/list_submissions.go`

**Request params:**

- none

**Query params:**

- `page?: number = 1`
- `limit?: number = 20`
- `status?: string`
- `language?: string`

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "limit": 20
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40000` invalid query parameters

**Frontend notes:**

- Phải gửi `credentials: "include"`.

#### GET /api/v1/my/submissions/{id}

**Description:**  
Lấy chi tiết submission của chính user.

**Auth required:** yes

**Handler:**  
`GetSubmissionHandler.HandleMy`  
File: `services/submission/internal/adapter/inbound/http/handler/submission/get_submission.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "id": 1,
    "problem_id": 1,
    "problem_name": "two-sum",
    "user_id": "string",
    "username": "string",
    "language": "GO",
    "status": "WRONG_ANSWER",
    "created_at": "string",
    "source_code": "string",
    "execution_time_ms": 12,
    "memory_used_kb": 1024,
    "compile_output": "string",
    "total_tests": 10,
    "failed_test_index": 3,
    "failed_test": {
      "test_index": 3,
      "status": "WRONG_ANSWER",
      "input": "string",
      "expected_output": "string",
      "actual_output": "string",
      "execution_time_ms": 1,
      "memory_used_kb": 128
    }
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40001` invalid URI params
- `40400` Submission not found
- `40300` forbidden

**Frontend notes:**

- Response detail hiện không trả full danh sách result của toàn bộ testcase.
- Hiện chỉ có `failed_test` summary.

#### GET /api/v1/admin/submissions/{id}

**Description:**  
Lấy submission detail cho admin.

**Auth required:** yes

**Handler:**  
`GetSubmissionHandler.HandleAdmin`  
File: `services/submission/internal/adapter/inbound/http/handler/submission/get_submission.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "id": 1,
    "problem_id": 1,
    "problem_name": "string",
    "user_id": "string",
    "username": "string",
    "language": "string",
    "status": "string",
    "created_at": "string",
    "source_code": "string",
    "execution_time_ms": 1,
    "memory_used_kb": 128,
    "compile_output": "string",
    "total_tests": 10,
    "failed_test_index": 1,
    "failed_test": {
      "test_index": 1,
      "status": "string",
      "input": "string",
      "expected_output": "string",
      "actual_output": "string",
      "execution_time_ms": 1,
      "memory_used_kb": 128
    }
  }
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40001` invalid URI params
- `40400` Submission not found

**Frontend notes:**

- Admin route tách riêng khỏi user route.

#### PUT /api/v1/admin/submissions/{id}/rejudge

**Description:**  
Queue lại submission để judge lại.

**Auth required:** yes

**Handler:**  
`RejudgeSubmissionHandler.Handle`  
File: `services/submission/internal/adapter/inbound/http/handler/submission/rejudge_submission.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20001`

```json
{
  "status": "success",
  "code": 20001,
  "msg": "Submission rejudge queued",
  "data": null
}
```

**Error cases:**

- `40100` unauthorized
- `40300` forbidden
- `40001` invalid URI params
- `40400` Submission not found

**Frontend notes:**

- Endpoint này queue job async, không trả judge result ngay.

### Contests

- Chưa tìm thấy contest route, DTO, entity, hoặc handler nào trong backend Go hiện tại.

### Users

- Chưa tìm thấy user-facing REST endpoint implemented trong service router.
- Có `User` entity và các DTO như `Profile` hoặc `UpdateUserRole`, nhưng chưa thấy router hiện tại map vào handler/use case tương ứng.

### Admin

- Các admin endpoint đã được liệt kê trong các section `Auth`, `Problems`, và `Submissions`.

### Other

#### GET /health

**Description:**  
Health check cho từng service.

**Auth required:** no

**Handler:**  
Inline handler trong từng router  
File: `services/auth/internal/adapter/inbound/http/router.go`, `services/problem/internal/adapter/inbound/http/router.go`, `services/submission/internal/adapter/inbound/http/router.go`

**Request params:**

- none

**Query params:** none

**Request body:** none

**Response:**

```json
{
  "status": "ok"
}
```

**Error cases:**

- none expected

**Frontend notes:**

- Endpoint này không dùng business response wrapper `{ status, code, msg, data }`.

#### GET /internal/v1/problems/{id}/testcases

**Description:**  
Internal route cho worker lấy metadata testcase của problem.

**Auth required:** no

**Handler:**  
`GetTestCaseForWorkerHandler.Handle`  
File: `services/problem/internal/adapter/inbound/http/handler/test_case/get_testcase_for_worker.go`

**Request params:**

- `id: number`

**Query params:** none

**Request body:** none

**Response:**

Code: `20000`

```json
{
  "status": "success",
  "code": 20000,
  "data": {
    "problem_id": 1,
    "test_count": 10,
    "version": "string",
    "zip_download_url": "string"
  }
}
```

**Error cases:**

- `40001` invalid URI params
- `40400` Test case not found

**Frontend notes:**

- Đây là service-to-service endpoint.
- Không dành cho browser frontend.

## Data types

### User

Entity hiện có:

```json
{
  "id": "string",
  "full_name": "string",
  "username": "string",
  "email": "string",
  "password": "hashed string",
  "role": "user | org_admin | org_member | org_contributor | global_contributor | global_moderator | super_admin",
  "rating": 0,
  "is_active": false,
  "created_at": "time",
  "updated_at": "time"
}
```

Frontend-usable DTO tìm thấy:

```json
{
  "username": "string",
  "email": "string",
  "rating": 0,
  "created_at": "time"
}
```

TypeScript helper:

```ts
export type UserProfile = {
  username: string
  email: string
  rating: number
  created_at: string
}
```

### Problem

Entity:

```json
{
  "id": 1,
  "title": "string",
  "title_slug": "string",
  "description": "string",
  "difficulty": "EASY | MEDIUM | HARD",
  "examples": [
    {
      "input": "string",
      "output": "string",
      "explanation": "string"
    }
  ],
  "constraints": "string",
  "hints": ["string"],
  "time_limit": 1,
  "memory_limit": 128,
  "author_id": "string",
  "is_hidden": true,
  "created_at": "time",
  "updated_at": "time"
}
```

Frontend response DTO:

```json
{
  "id": 1,
  "slug": "string",
  "title": "string",
  "description": "string",
  "difficulty": "EASY | MEDIUM | HARD",
  "examples": [
    {
      "input": "string",
      "output": "string",
      "explanation": "string"
    }
  ],
  "constraints": "string",
  "hints": ["string"],
  "time_limit": 1,
  "memory_limit": 128,
  "author_id": "string",
  "is_hidden": false,
  "created_at": "string"
}
```

TypeScript helper:

```ts
export type ProblemExample = {
  input: string
  output: string
  explanation?: string
}

export type Problem = {
  id: number
  slug: string
  title: string
  description: string
  difficulty: "EASY" | "MEDIUM" | "HARD"
  examples?: ProblemExample[]
  constraints?: string
  hints?: string[]
  time_limit: number
  memory_limit: number
  author_id?: string
  is_hidden?: boolean
  created_at: string
}
```

### Submission

Entity:

```json
{
  "id": 1,
  "problem_id": 1,
  "problem_name": "string",
  "user_id": "string",
  "username": "string",
  "language": "C | CPP | JAVA | PYTHON | GO | JAVASCRIPT",
  "source_code": "string",
  "status": "PENDING | JUDGING | ACCEPTED | WRONG_ANSWER | TIME_LIMIT_EXCEEDED | MEMORY_LIMIT_EXCEEDED | RUNTIME_ERROR | COMPILATION_ERROR | SYSTEM_ERROR",
  "execution_time": 1,
  "memory_used": 128,
  "compile_output": "string",
  "created_at": "time",
  "updated_at": "time"
}
```

Frontend list/detail DTO:

```json
{
  "id": 1,
  "problem_id": 1,
  "problem_name": "string",
  "user_id": "string",
  "username": "string",
  "language": "string",
  "status": "string",
  "created_at": "string",
  "source_code": "string",
  "execution_time_ms": 1,
  "memory_used_kb": 128,
  "compile_output": "string",
  "total_tests": 10,
  "failed_test_index": 1,
  "failed_test": {
    "test_index": 1,
    "status": "string",
    "input": "string",
    "expected_output": "string",
    "actual_output": "string",
    "execution_time_ms": 1,
    "memory_used_kb": 128
  }
}
```

TypeScript helper:

```ts
export type SubmissionStatus =
  | "PENDING"
  | "JUDGING"
  | "ACCEPTED"
  | "WRONG_ANSWER"
  | "TIME_LIMIT_EXCEEDED"
  | "MEMORY_LIMIT_EXCEEDED"
  | "RUNTIME_ERROR"
  | "COMPILATION_ERROR"
  | "SYSTEM_ERROR"

export type SubmissionLanguage =
  | "C"
  | "CPP"
  | "JAVA"
  | "PYTHON"
  | "GO"
  | "JAVASCRIPT"
```

### TestCase

Entity:

```json
{
  "id": 1,
  "problem_id": 1,
  "zip_object_key": "string",
  "test_count": 10,
  "version": "string",
  "created_at": "time"
}
```

Frontend-related DTO:

```json
{
  "test_count": 10,
  "version": "string"
}
```

Internal worker DTO:

```json
{
  "problem_id": 1,
  "test_count": 10,
  "version": "string",
  "zip_download_url": "string"
}
```

### Language

Supported values:

- `C`
- `CPP`
- `JAVA`
- `PYTHON`
- `GO`
- `JAVASCRIPT`

### JudgeResult

Internal judge message hoặc stored-result related types tìm thấy:

```json
{
  "submission_id": 1,
  "status": "string",
  "compile_output": "string",
  "execution_time": 1,
  "memory_used": 128,
  "test_cases": [
    {
      "index": 1,
      "status": "string",
      "actual_output": "string",
      "input": "string",
      "expected_output": "string",
      "execution_time": 1,
      "memory_used": 128
    }
  ]
}
```

Stored per-test result:

```json
{
  "submission_id": 1,
  "test_index": 1,
  "status": "ACCEPTED | WRONG_ANSWER | TIME_LIMIT_EXCEEDED | MEMORY_LIMIT_EXCEEDED | RUNTIME_ERROR | SYSTEM_ERROR",
  "actual_output": "string",
  "input": "string",
  "expected_output": "string",
  "execution_time": 1,
  "memory_used": 128
}
```

## Unknown / needs confirmation

Các endpoint sau có trong gateway config hoặc tài liệu, nhưng chưa thấy route hoặc handler implemented trong service router hiện tại:

- `POST /api/v1/auth/verify-forgot-password`
- `GET /api/v1/auth/profile/{username}`
- `GET /api/v1/auth/profile`
- `PUT /api/v1/auth/admin/{username}/role`
- `GET /api/v1/admin/problems/{id}/testcases`
- `PUT /api/v1/admin/testcases/{id}`
- `DELETE /api/v1/admin/testcases/{id}`

Trạng thái đề xuất:

- `Mock allowed until backend ready`

Các điểm cần confirm:

- `problem_name` trong `POST /api/v1/submissions` có phải thực chất là problem slug không. Pipeline judge hiện map field này vào `problem_slug`.
- `GET /api/v1/submissions` hiện đang public, nhưng cần confirm đây có phải behavior chủ đích lâu dài hay không.
- `GET /api/v1/auth/profile` và `GET /api/v1/auth/profile/{username}` endpoint nào mới là path đúng nếu backend hoàn thiện.
- Admin testcase management ngoài upload hiện chưa thấy implemented trong problem service.
- Frontend có cần full testcase-by-testcase result cho submission detail hay `failed_test` summary là đủ.

## Change policy

- Chỉ update file này khi backend thay đổi thật.
- Không đổi method, path, auth behavior, request shape, response shape nếu không có bằng chứng từ backend.
- Không đổi public endpoint thành protected hoặc ngược lại nếu backend chưa đổi.
- Không xóa endpoint đang có trong contract chỉ vì frontend chưa dùng tới.
- Khi phát hiện mismatch giữa frontend type và backend response, ưu tiên cập nhật hoặc báo lại dựa trên backend source, không tự suy diễn.
