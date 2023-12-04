---
id: bundle
title: Bundle Service
sidebar_label: Data Bundles
slug: /getting-started/examples
---

We introduce a new services to model how relationships and attributes are created and deleted when actions occur on resources. Managing these data centrally promises better transparency and consistent modeling.

## How Bundles Works 

Let's examine how Bundles operates with basic example. 

Let's say you want to model how data will be created when an organization created in your application. For this purpose, you can utilize the [WriteBundle](./api-overview/bundle/write-bundle.md) API endpoint. This API enables users to define or update data bundles, each distinguished by a unique name.

Here's an example body for WriteBundle in this scenario:

```json
"bundles": [
        {
            "name": "organization_created"
            "arguments": [
                "creatorID",
                "organizationID"
            ],
            "operations": [
                {
                    "relationships_write": [
                        "organization:{{.organizationID}}#admin@user:{{.creatorID}}",
                        "organization:{{.organizationID}}#manager@user:{{.creatorID}}",
                    ],
                    "attributes_write": [
                        "organization:{{.organizationID}}$public|boolean:false",
                    ],
                },
            ],
        },
    ],
```

Operations represent actions that can be performed on relationships and attributes, such as adding or deleting relationships when certain events occur.

Let's say user:564 creates an organization:789 in your application. According to your authorization logic, this will result in the creation of several authorization data, including relational tuples and attributes, respectively.

* organization:789#admin@user:564
* organization:789#manager@user:564
* organization:789$public|boolean:false

Instead of using the [WriteData](./api-overview/data/write-data.md) endpoint, you can utilize [RunBundle](./api-overview/data/run-bundle.md) to create this data by simply providing specific identifiers.

An example request of [RunBundle](./api-overview/data/run-bundle.md) for this scenario: 

```json
POST /bundle
BODY 
{
   "name": "project_created",
   "arguments": {
       "creatorID": "564",
       "organizationID": "789",
    }
}
```

This will result in the creation of the following data in Permify:

* organization:789#admin@user:564
* organization:789#manager@user:564
* organization:789$public|boolean:false

## Endpoints

- [WriteBundle](./api-overview/bundle/write-bundle.md)
- [RunBundle](./api-overview/data/run-bundle.md)
- [DeleteBundle](./api-overview/bundle/delete-bundle.md)
- [ReadBundle](./api-overview/bundle/read-bundle.md)