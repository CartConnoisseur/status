# status

Just something I threw together real quick for an incredibly specific and niche use case (displaying silly little messages on my girlfriend's dashboard)

## Usage
- Create the directory `~/.status`
- Run the server (see compilation instructions below)
- Make requests to `/users` with given admin credentials to create users
    - With cURL: `curl -u admin:password -X POST -H "Content-Type: application/json" -d '{"username": "<username>", "password": "<password>"}' <address>/users`
- Set a user's message
    - With cURL: `curl -u username:password -d 'message' -X POST <address>/status`
- Read the message
    - With cURL: `curl -u username:password <address>`

If that seems incredibly janky, that's because it is :3

## Compiling
Requirements
- go >= 1.22.5

Instructions:
1. Clone the repo
```bash
git clone https://github.com/CartConnoisseur/status
cd status
```
2. Install dependencies
```bash
go mod tidy
```

3. Compile
```bash
go build -o status
```

The server can then be run via `./status <port>`.
