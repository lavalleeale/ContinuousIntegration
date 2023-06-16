openssl genrsa -out CA.key 2048
openssl req -x509 -new -nodes -key CA.key -subj "/C=US/CN=Registry Auth CA" -sha256 -days 1825 -out CA.pem

openssl genrsa -out registry.key 2048
openssl req -new -key registry.key -subj "/C=US/CN=registry" -config registry.conf -out registry.csr
openssl x509 -req -in registry.csr -CA CA.pem -CAkey CA.key -extfile registry.conf -outform pem -out registry.crt -days 825 -sha256 -extensions v3_req

openssl genrsa -out auth.key 2048
openssl req -new -key auth.key -subj "/C=US/CN=auth" -config auth.conf -out auth.csr
openssl x509 -req -in auth.csr -CA CA.pem -CAkey CA.key -extfile auth.conf -outform pem -out auth.crt -days 825 -sha256 -extensions v3_req

cp CA.pem ../../../images/*/
