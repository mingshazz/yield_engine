# Yield Engine
This project provides a RESTful API built with Go and Gin to upload and query CSV data. The API connects to a MySQL database to store and retrieve data.

## 🖥️ Run Locally
To run the application locally with MySQL running on the machine, follow these steps:
1. Install MySQL (if not already installed)

If MySQL is not installed yet, install it using Homebrew:
```bash
brew update
brew install mysql
brew services start mysql
```
2. Create a MySQL Database

Open a MySQL terminal:
```bash
mysql -u root
```
Create the necessary database and user:
```sql
CREATE DATABASE csv_data;
CREATE USER 'go_user'@'localhost' IDENTIFIED BY 'go_pass';
GRANT ALL PRIVILEGES ON csv_data.* TO 'go_user'@'localhost';
FLUSH PRIVILEGES;
```
3. Set Environment Variables

Either create a .env file, or export the environment variables manually.

Option 1: Create a .env file
Create a .env file in the project directory with the following content:
```go
DB_USER=go_user
DB_PASS=go_pass
DB_HOST=localhost
DB_NAME=csv_data
```
Option 2: Export Environment Variables
Instead of using a .env file, set the environment variables manually:
```bash
export DB_USER=go_user
export DB_PASS=go_pass
export DB_HOST=localhost
export DB_NAME=csv_data
```
4. Install Go Dependencies

Run the following command to install the required dependencies:
```bash
go mod tidy
```
5. Run the Application

Start the Go application:
```bash
go run main.go
```
The server will now be running on http://localhost:8080.

6. Test the API Locally

Test the API using Postman or curl.
Upload CSV:
```bash
curl -X POST -F "file=@filename.csv" http://localhost:8080/data
```
Query data:
```bash
curl "http://localhost:8080/data?symbol=BTC&page=1&limit=10"
```

# 🐳 Run with Docker (Using Docker Compose)
To run the application and MySQL in Docker containers, follow these steps:
1. Set Up the Project

Ensure there are the following files in the project directory:
+ main.go (Go code)
+ Dockerfile (to build the Go app)
+ docker-compose.yml (to configure the Docker services)
+ .env (to store environment variables)
2. Set Up Environment Variables

There is a .env file in the project directory with the following content:
```go
DB_USER=go_user
DB_PASS=go_pass
DB_HOST=db    # Use the service name for MySQL in Docker Compose
DB_NAME=csv_data
```
3. Dockerize the Application

There is a Dockerfile and docker-compose.yml for Docker compose configuration.

4. Build and Start the Containers

From the root directory of the project, run the following command:
```bash
docker-compose up --build
```
This will build both the Go app container and the MySQL container, and then start the services.

5. Test the API in Docker

Test the API using Postman or curl.
Upload CSV:
```bash
curl -X POST -F "file=@filename.csv" http://localhost:8080/data
```
Query Data:
```bash
curl "http://localhost:8080/data?symbol=BTC&page=1&limit=10"
```

# 📝 API Endpoints
1. POST /data

Uploads a CSV file containing trading data. The CSV file must have the following columns:

UNIX: The UNIX timestamp of the trade.

SYMBOL: The trading symbol (e.g., BTC/USD).

OPEN: The opening price of the trade.

HIGH: The highest price during the trade.

LOW: The lowest price during the trade.

CLOSE: The closing price of the trade.

Example:
```bash
curl -X POST -F "file=@filename.csv" http://localhost:8080/data
```
2. GET /data

Retrieves data from the database. Supports pagination and searching by symbol.

Query Parameters:

symbol: The trading symbol (e.g., BTC).

page: The page number (for pagination).

limit: The number of results per page.

Example:
```bash
curl "http://localhost:8080/data?symbol=BTC&page=1&limit=10"
```

# 📝 Troubleshooting
Error: "failed to initialize database": Ensure MySQL credentials and host in the .env file are correct. If using Docker, make sure the service name for MySQL (db) is specified as the DB_HOST.

Error: "MySQL container not starting": Check the MySQL logs by running docker-compose logs db to troubleshoot the issue.