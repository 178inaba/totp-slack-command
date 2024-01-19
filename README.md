# TOTP Slack Command

## Usage

```console
$ gcloud functions deploy GenerateTOTP \
  --gen2 \
  --trigger-http \
  --allow-unauthenticated \
  --runtime go121 \
  --env-vars-file .env.yaml \
  --region <region> \
  --project <project-id>
```

## License

[MIT](LICENSE)

## Author

Masahiro Furudate (a.k.a. [178inaba](https://github.com/178inaba))  
<178inaba.git@gmail.com>
