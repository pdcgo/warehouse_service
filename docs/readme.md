# Warehouse Service
This Is part of Submodule of [Warehouse Infra](https://github.com/pdcgo/warehouse_infra). In Warehouse Infra this is live in folder `./warehouse_service`.<br>
This Service planned and Intended for replacing legacy Warehouse System that exists in Warehouse Infra. Its planned for microservice and planned to more dependentless, separating domain purpose for better developing big and complex system that exists in Warehouse Infra.<br>
For now its just be candidate for Take over Warehouse System In Warehouse Infra legacy.
Status for this development is still in progress and not completely take over legacy system.

1. for database schema related, read this [Database Schema](database-schema.md).

## Authentication & Authorization
1. Use v2 roling system. not legacy system.

## Connect RPC Spec
`WarehouseService` heavyly depend `connect-rpc` to serve and creating apis and grpc. Why we use `connectrpc` because its can be two mode as pure grpc and grpc-web that interact like web. And also supported http2. This service have several rpc:

1. Warehouse Management related RPC
    - Create Warehouse that named `WarehouseCreate`
    - Delete Warehouse that named `WarehouseDelete`
    - List Warehouse that named `WarehouseList`
    - Update Warehouse that named `WarehouseUpdate`
    - Detail warehouse that named `WarehouseDetail`



### Warehouse Management RPC
1. warehouse management is for admin team.
2. `WarehouseList` and `WarehouseDetail` available for all authenticated user.