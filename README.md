### Functional Requirements
1. **Off-Chain Indexing Service**: Build an off-chain indexing service that listens for events related to USDC and ETH tokens, specifically for the "Borrow" and "Mint" events.
   
2. **Event Handling**:
   - Capture and process "Borrow" events, issuing 2 points for every 1 unit of currency borrowed, calculated every 10 minutes.
   - Capture and process "Mint" events, issuing 1 point for every 1 unit of currency minted, calculated every 10 minutes.

### Non Requirements
1. **Event Tracking**: The server does not log the total number of mint or borrow events. For example, if a user mints 10 USDC, redeems (or repays) $5, and then redeems another $5, the point calculation will be finalized at the first redemption, and the second redemption will not affect the point calculation.

### Non-Functional Requirements
1. Transaction Replay Capability: The server should be able to replay transactions even if it has gone down, ensuring that no events or points are lost during downtime.

### Hot to run the project
```
$ make up-db // create postgres container
$ make migrate-up
$ make run
```