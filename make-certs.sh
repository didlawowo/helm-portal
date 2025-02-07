# Générer la clé privée pour le CA
openssl genrsa -out src/certs/ca.key 2048

# Générer le certificat CA
openssl req -x509 -new -nodes -key src/certs/ca.key -sha256 -days 365 -out src/certs/ca.crt \
    -subj "/CN=Test CA"

# Générer la clé privée du service
openssl genrsa -out src/certs/service.key 2048

# Générer la demande de certificat (CSR) pour le service
openssl req -new -key src/certs/service.key -out src/certs/service.csr \
    -subj "/CN=Test Service"

# Signer le certificat du service avec le CA
openssl x509 -req -in src/certs/service.csr -CA src/certs/ca.crt -CAkey src/certs/ca.key \
    -CAcreateserial -out src/certs/service.crt -days 365 -sha256
