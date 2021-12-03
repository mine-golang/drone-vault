
#!/bin/bash
version=$1
dockerfile=$2
sudo docker build -f ${dockerfile} -t wssio/drone-vault:${version} .
sudo docker push wssio/drone-vault:${version}
if [ $? -eq 0 ]; then
 echo "push Success"
else 
 echo "push failed"
fi