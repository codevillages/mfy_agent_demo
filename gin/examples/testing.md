# Gin v1.7.7 测试示例（httptest）

## 单元测试 Handler
```go
func TestGetUser(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.GET("/users/:id", h.GetUser)

    req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)
    require.Equal(t, http.StatusOK, w.Code)
    require.Contains(t, w.Body.String(), `"id":"123"`)
}
```

## 带中间件的测试（RequestID/日志）
```go
r := gin.New()
r.Use(requestIDMiddleware(), logMiddleware())
r.POST("/users", h.CreateUser)
body := bytes.NewBufferString(`{"name":"alice","email":"a@b.com"}`)
req := httptest.NewRequest(http.MethodPost, "/users", body)
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()
r.ServeHTTP(w, req)
require.Equal(t, http.StatusCreated, w.Code)
```

## Mock 下游依赖
```go
svc := new(MockUserService)
svc.On("Get", mock.Anything, "123").Return(User{ID: "123"}, nil)
h := NewHandler(svc)
```

## 性能/路由回归测试（示例）
```go
func BenchmarkList(b *testing.B) {
    gin.SetMode(gin.TestMode)
    r := setupRouter() // 生产同配置
    for i := 0; i < b.N; i++ {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&page_size=10", nil)
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        if w.Code != http.StatusOK { b.FailNow() }
    }
}
```

## 小贴士
- 测试前设置 `gin.SetMode(gin.TestMode)` 避免日志干扰。
- 用 `httptest.NewRecorder()` + `ServeHTTP` 走完整链路（含中间件）。
- 需要上下文值时，提前在请求头写 `X-Request-ID` 或手动 `c.Set`（在中间件里）。
- 结合 mock（gomock/testify）隔离外部依赖；必要时用 `httptest.Server` 做端到端测试。
