# wwfc
WiiLink Wi-Fi Connection aims to be an open source server replacement for Nintendo Wi-Fi Connection. Currently, a work in progress.

## Current Support
- Matchmaking (No server sorting yet)
- Adding Friends
- Downloading and Uploading Ghosts
- Restored Worldwide Ghost Race mode functionality (great for small/new databases) 

## Setup
You will need:
- PostgreSQL

1. Create a PostgreSQL database. Note the database name, username, and password.
2. Use the `schema.sql` found in the root of this repo and import it into your PostgreSQL database.
3. Copy `config-example.xml` to `config.xml` and insert all the correct data.
4. Run `go build`. The resulting executable `wwfc` is the executable of the server.
5. Add a `payload` folder, containing the `private-key.pem`, `stage1.bin`, and `binaries` folder with each binary for each compatible game.

Beware, not every wiilink24-based payload will work interchangeably. It's highly recommended that you build your own set of `payloads`, `gecko codes`, and `private keys`, or use (at least) a compatible `payload`.
