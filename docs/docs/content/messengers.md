# Messengers

listmonk supports multiple custom messaging backends in additional to the default SMTP e-mail backend, enabling not just e-mail campaigns, but arbitrary message campaigns such as SMS, FCM notifications etc.

A *Messenger* is a web service that accepts a campaign message pushed to it as a JSON request, which the service can in turn broadcast as SMS, FCM etc. Messengers are registered in the *Settings -> Messengers* UI, and can be selected on individual campaigns.

Messengers support optional BasicAuth authentication. `Plain text` format for campaign content is ideal for messengers such as SMS and FCM.

When a campaign starts, listmonk POSTs messages in the following format to the selected messenger's endpoint. The endpoint should return a `200 OK` response in case of a successful request.

The address required to broadcast the message, for instance, a phone number or an FCM ID, is expected to be stored and relayed as [subscriber attributes](concepts.md/#attributes). 

```json
{
	"subject": "Welcome to listmonk",
	"body": "The message body",
	"content_type": "plain",
	"recipients": [{
		"uuid": "e44b4135-1e1d-40c5-8a30-0f9a886c2884",
		"email": "anon@example.com",
		"name": "Anon Doe",
		"attribs": {
			"phone": "123123123",
			"fcm_id": "2e7e4b512e7e4b512e7e4b51",
			"city": "Bengaluru"
		},
		"status": "enabled"
	}],
	"campaign": {
		"uuid": "2e7e4b51-f31b-418a-a120-e41800cb689f",
		"name": "Test campaign",
		"tags": ["test-campaign"]
	}
}
```

## Messenger implementations

Following is a list of HTTP messenger servers that connect to various backends.

| Name                                                                                 | Backend          |
|:-------------------------------------------------------------------------------------|:-----------------|
| [listmonk-messenger](https://github.com/joeirimpan/listmonk-messenger)               | AWS Pinpoint SMS |
| [listmonk-verimor-gateway](https://github.com/antandros/listmonk-verimor-gateway)    | Verimor          |
| [listmonk-mailersend](https://github.com/tkawczynski/listmonk-mailersend)            | Mailersend       |
| [listmonk-novu-messenger](https://github.com/Codepowercode/listmonk-novu-messenger)  | Novu             |
| [listmonk-push-messenger](https://github.com/shyamkrishna21/listmonk-push-messenger) | Google FCM       |
