# Word Dictionary Learning Site

A web application for learning and managing vocabulary words, built with Go and MySQL.

## Features

- User authentication (login/logout)
- Password change functionality
- Word management (add, edit, delete)
- Phonetic notation display
- Audio playback support
- Real-time word editing
- Responsive design

## Prerequisites

- Go 1.16 or higher
- MySQL 5.7 or higher
- Modern web browser

## Setup

1. Create a MySQL database:
```sql
CREATE DATABASE word_dict;
```

2. Configure MySQL connection in main.go (if needed):
```go
db, err = gorm.Open("mysql", "root:root@tcp(127.0.0.1:3306)/word_dict?charset=utf8&parseTime=True&loc=Local")
```

3. Install dependencies:
```bash
go mod tidy
```

4. Run the application:
```bash
go run main.go
```

5. Access the application at http://localhost:8080

## Default User

- Username: root
- Password: 1234567

## API Endpoints

- POST /login - User login
- POST /change-password - Change user password
- GET /words - Get all words
- POST /words - Add a new word
- PUT /words/:id - Update a word
- DELETE /words/:id - Delete a word

## Project Structure

```
word_dict/
├── main.go          # Main application file
├── templates/       # HTML templates
│   └── index.html   # Main page template
├── static/         # Static files
│   ├── style.css   # CSS styles
│   └── app.js      # Vue.js application code
└── go.mod          # Go module file
```
