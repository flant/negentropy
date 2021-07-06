set -exu

apk add python3

mkdir -p /tmp/build && \
cd /tmp/build && \
wget https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.tar.gz && \
tar -C /usr/local -xvf google-cloud-sdk.tar.gz && \
/usr/local/google-cloud-sdk/install.sh --quiet && \
cd /tmp && \
rm -rf /tmp/build

ln -s /usr/local/google-cloud-sdk/bin/gcloud /bin/gcloud
ln -s /usr/local/google-cloud-sdk/bin/gsutil /bin/gsutil
