# Integrating with external systems

In many environments, a mailing list manager's subscriber database is not run independently but as a part of an existing customer database or a CRM. There are multiple ways of keeping listmonk in sync with external systems.

## Using APIs

The [subscriber APIs](apis/subscribers.md) offers several APIs to manipulate the subscribers database, like addition, updation, and deletion. For bulk synchronisation, a CSV can be generated (and optionally zipped) and posted to the import API.

## Interacting directly with the DB

listmonk uses tables with simple schemas to represent subscribers (`subscribers`), lists (`lists`), and subscriptions (`subscriber_lists`). It is easy to add, update, and delete subscriber information directly with the database tables for advanced usecases. See the [table schemas](https://github.com/knadh/listmonk/blob/master/schema.sql) for more information.
