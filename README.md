# mars-property-chaincode

This chaincode is used to register properties on mars and assign an owner to each property. The identity of the owner will be verified using the "identity" chaincode. This chaincode will be deployed on the "property" channel by the property authority.

## Install and Instantiate 

First ssh into the EC2 that's running the container. Then access to shell of the container of property authority using this command: `docker exec -i -t container_id /bin/bash`. 

Then follow this steps to install and instantiate the chaincode:

1. Create the directory using `mkdir /opt/gopath/src && mkdir /opt/gopath/src/github.com` if doesn't exist.
2. Clone the chaincode repo using the command `cd /opt/gopath/src/github.com && git clone https://github.com/narayanprusty/mars-property-chaincode.git`
3. Install using this command: `peer chaincode install -n property -v 1.0 -p github.com/mars-property-chaincode`
4. Command to instantiate the chaincode: `peer chaincode instantiate -o $ORDERER_URL -C property -n property -v 1.0 -c '{"Args":[]}' --cafile /home/crypto/managedblockchain-tls-chain.pem --tls`

