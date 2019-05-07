const shim = require('fabric-shim');
const util = require('util');
const ethGSV = require('ethereum-gen-sign-verify');

var Chaincode = class {

  // Initialize the chaincode
  async Init(stub) {
    try {
      await stub.putState("propertyAuthority", Buffer.from(stub.getCreator().mspid));
      return shim.success();
    } catch (err) {
      return shim.error(err);
    }
  }

  async Invoke(stub) {
    let ret = stub.getFunctionAndParameters();
    let method = this[ret.fcn];
    
    if (!method) {
      console.log('no method of name:' + ret.fcn + ' found');
      return shim.error();
    }

    try {
      let payload = await method(stub, ret.params);
      return shim.success(payload);
    } catch (err) {
      console.log(err);
      return shim.error(err);
    }
  }

  //get the mspid of the property authority
  async getCreatorIdentity(stub) {
    let creatorIdentity = await stub.getState('propertyAuthority');

    if (!creatorIdentity) {
      throw new Error("Creator identity not found");
    }

    return creatorIdentity;
  }

  async addProperty(stub, args) {
    if (args.length < 3) {
      throw new Error('Incorrect number of arguments.');
    }

    let id = args[0];
    let location = args[1];
    let ownerId = args[2];
    let createdOn = stub.getTxTimestamp().seconds.low;

    let property = {
      info: {
        location, createdOn, ownerId
      },
      history: []
    }

    if((await stub.getState('propertyAuthority')).toString() === stub.getCreator().mspid) {
      if(!(await stub.getState(`property_${id}`)).toString()) {
        await stub.putState(`property_${id}`, Buffer.from(JSON.stringify(property)))
      } else {
        throw new Error('Property already exists');
      }
    } else {
      throw new Error('You don\'t have permission to add property');
    }
  }

  async getProperty(stub, args) {
    if (args.length < 1) {
      throw new Error('Incorrect number of arguments.');
    }

    let property = await stub.getState(`property_${args[0]}`);
    if(!property) {
      throw new Error('Property not found');
    }

    return property;
  }

  async transferProperty(stub, args) {
    if (args.length < 4) {
      throw new Error('Incorrect number of arguments.');
    }   

    let id = args[0]
    let message = JSON.parse(args[1])
    let signature = JSON.parse(args[2])
    let identityChannelName = args[3]

    let property = (await stub.getState(`property_${id}`)).toString()

    if(property) {
      property = JSON.parse(property);

      let result = await stub.invokeChaincode('identity', [
        Buffer.from('getIdentity'),
        Buffer.from(property.info.ownerId)
      ], identityChannelName)

      if(result.status !== 200) {
        throw new Error('Internal transaction failed');
      }
  
      let publicKey = JSON.parse(result.payload.toString('utf8')).publicKey;

      if(message.action !== 'transfer') {
        throw new Error('Permission invalid');
      }

      let isValid = ethGSV.verify(JSON.stringify(message), signature, publicKey);

      if(!isValid) {
        throw new Error('Signature invalid');
      }

      property.history.push({
        action: 'transfer',
        previousOwner: property.info.ownerId,
        newOwner: message.newOwner
      });

      property.info.ownerId = message.newOwner;

      await stub.putState(`property_${id}`, Buffer.from(JSON.stringify(property)))
    } else {
      throw new Error('Property not found.');
    }
  }
};

shim.start(new Chaincode());
