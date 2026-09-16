#!/usr/bin/env bash
set -euo pipefail

# Generate real random values (no template placeholder that never gets replaced)
DB_PASS="$(openssl rand -hex 16)"
API_SECRET="$(openssl rand -hex 16)"

cat > .env << EOF
POSTGRES_USER=wiilink
POSTGRES_PASSWORD=${DB_PASS}
POSTGRES_DB=wwfc
EOF

sed -i -E "s#<password>.*</password>#<password>${DB_PASS}</password>#" config.xml
sed -i -E "s#<apiSecret>.*</apiSecret>#<apiSecret>${API_SECRET}</apiSecret>#" config.xml

echo "Done. .env and config.xml now contain synchronised, random values."
echo "Do NOT commit or share the password and API secret."
