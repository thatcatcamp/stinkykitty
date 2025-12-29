# Security Audit Report - Stinky Kitty
**Date:** December 29, 2025
**Auditor:** Gemini CLI Agent

## 1. Executive Summary
The Stinky Kitty application (a multi-tenant CMS) has a solid foundation with some modern security practices in place (bcrypt hashing, parameterized SQL queries, random filename generation). However, several critical and medium-risk vulnerabilities were identified that should be addressed before a production release. The most significant risks are the lack of CSRF protection and potential file upload vulnerabilities.

## 2. Detailed Findings

### 2.1 Authentication & Session Management

*   **[HIGH] Missing CSRF Protection**
    *   **Description:** The application uses cookies for authentication (`stinky_token`) but does not appear to implement Cross-Site Request Forgery (CSRF) protection tokens for state-changing operations (e.g., creating/deleting sites, updating settings).
    *   **Impact:** An attacker could trick an authenticated admin into performing unwanted actions without their knowledge.
    *   **Recommendation:** Implement a CSRF middleware (e.g., `gorilla/csrf` or `gin-contrib/csrf` depending on the framework) that validates a token in headers/forms for all non-GET requests.

*   **[MEDIUM] Insecure Cookie Attributes**
    *   **Description:** The session cookie is explicitly set with `Secure: false` in `internal/handlers/admin.go`.
    *   **Impact:** Cookies can be transmitted over unencrypted HTTP connections, making them vulnerable to interception (Man-in-the-Middle attacks).
    *   **Recommendation:** Set `Secure: true` when the application is running in a production environment (detect via config/env var). Ensure `SameSite` is set to `Lax` or `Strict`.

*   **[LOW] Password Hashing**
    *   **Description:** Passwords are hashed using `bcrypt` with a cost of 12.
    *   **Status:** **Secure.** This is industry standard.

### 2.2 Input Validation & Injection

*   **[LOW] SQL Injection**
    *   **Description:** Database interactions in `internal/db` utilize GORM with parameterized queries.
    *   **Status:** **Secure.** Standard ORM usage effectively prevents SQL injection.

*   **[LOW/MEDIUM] Cross-Site Scripting (XSS)**
    *   **Description:** The `internal/blocks/renderer.go` handles content rendering.
        *   Most blocks (Text, Heading, Quote) use `html.EscapeString`.
        *   The "Columns" block uses `bluemonday.UGCPolicy()` to sanitize HTML.
    *   **Status:** **Mostly Secure.** The use of `bluemonday` is a strong positive signal. Ensure that the "Video" block's URL parsing is strictly validated to prevent arbitrary iframe injection.

### 2.3 File Uploads

*   **[HIGH] Weak File Type Validation**
    *   **Description:** In `internal/uploads/uploader.go`, the `IsImageFile` function validates files based solely on their file extension. It does not check the file's "Magic Bytes" (MIME sniffing).
    *   **Impact:** An attacker could rename a malicious script (e.g., `exploit.sh` or `webshell.php`) to `image.jpg` to bypass the check. If the server is misconfigured to execute files in the upload directory, this leads to Remote Code Execution (RCE).
    *   **Recommendation:** Use `http.DetectContentType` or a library like `h2non/filetype` to validate the actual file content before saving.

*   **[MEDIUM] Missing File Size Limits**
    *   **Description:** `SaveUploadedFile` does not appear to enforce a file size limit explicitly (though the HTTP handler might).
    *   **Impact:** Denial of Service (DoS) via disk space exhaustion.
    *   **Recommendation:** Enforce a strict limit (e.g., 5MB) on the `multipart.FileHeader` size or use `http.MaxBytesReader` in the handler.

### 2.4 Configuration & Deployment

*   **[MEDIUM] Missing HTTP Security Headers**
    *   **Description:** The middleware chain does not set standard security headers.
    *   **Impact:** Increases attack surface.
    *   **Recommendation:** Add middleware to set:
        *   `Strict-Transport-Security` (HSTS)
        *   `X-Content-Type-Options: nosniff`
        *   `X-Frame-Options: DENY` (or `SAMEORIGIN`)
        *   `Content-Security-Policy` (CSP)

## 3. Prioritized Remediation Plan

1.  **Immediate:** Implement CSRF protection.
2.  **Immediate:** Fix file upload validation (Magic Bytes check).
3.  **Short-term:** Enable `Secure` cookies for production builds.
4.  **Short-term:** Add Security Headers middleware.
