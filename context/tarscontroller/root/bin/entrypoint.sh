#! /bin/bash

declare OPENSSL_WORK_DIR=/etc/tarscontroller-cert
declare CA_CRT_FILE=${OPENSSL_WORK_DIR}/ca.crt
declare CA_KEY_FILE=${OPENSSL_WORK_DIR}/ca.key
declare TLS_KEY_FILE=${OPENSSL_WORK_DIR}/tls.key
declare TLS_CSR_FILE=${OPENSSL_WORK_DIR}/tls.csr
declare TLS_CRT_FILE=${OPENSSL_WORK_DIR}/tls.crt
declare NAMESPACE_FILE=/var/run/secrets/kubernetes.io/serviceaccount/namespace

SERVICE_NAME="tars-webhook-service"
NAMESPACE_VALUE=$(cat ${NAMESPACE_FILE})

TLS_CN=${SERVICE_NAME}.${NAMESPACE_VALUE}.svc

if [ -z "$NAMESPACE_VALUE" ]; then
  echo "read \$NAMESPACE_FILE error"
  exit 255
fi

if ! cd $OPENSSL_WORK_DIR; then
  echo "cd \$OPENSSL_WORK_DIR error"
  exit 255
fi

#  generate $TLS_KEY_FILE
if ! openssl genrsa -out $TLS_KEY_FILE 2048; then
  echo "generate \$TLS_KEY_FILE error"
  exit 255
fi
#

# generate $TLS_CSR_FILE
if ! openssl req -new -key $TLS_KEY_FILE -out $TLS_CSR_FILE -extensions "v3_req" -subj "/CN=${TLS_CN}" -reqexts SAN -config <(cat /etc/ssl/openssl.cnf <(printf "[SAN]\nsubjectAltName=DNS:%s" "${TLS_CN}")); then
  echo "generate \$TLS_CSR_FILE error"
  exit 255
fi
#
# generate $TLS_CRT_FILE
if ! openssl x509 -days 365 -req -in $TLS_CSR_FILE -CA $CA_CRT_FILE -CAkey $CA_KEY_FILE -CAcreateserial -out $TLS_CRT_FILE -extfile <(printf "subjectAltName=DNS:%s" "${TLS_CN}"); then
  echo "generate \$TLS_CRT_FILE error"
  exit 255
fi
#

# verify $TLS_CRT_FILE
if ! openssl verify -CAfile $CA_CRT_FILE $TLS_CRT_FILE; then
  echo "verify \$TLS_CRT_FILE error"
  exit 255
fi

TARSCONTROLLER_EXECUTION_FILE="/usr/local/app/tars/tarscontroller/bin/tarscontroller"

chmod +x ${TARSCONTROLLER_EXECUTION_FILE}
exec ${TARSCONTROLLER_EXECUTION_FILE}
