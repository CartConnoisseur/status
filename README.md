# status

Just something I threw together real quick for an incredibly specific and niche use case (displaying silly little messages on my girlfriend's dashboard)

## Usage
- Create `~/.status/auth`
- Run the server
- Make requests to `/generate-hash` to create user and admin hashes
    - With cURL: `curl -u username:password <address>/generate-hash`
- Put generated hashes into `~/.status/auth`
- Restart the server
- Set a message
    - With cURL: `curl -u username:password -d 'message' -X POST <address>/update`
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

The server can then be run via `./status`.
