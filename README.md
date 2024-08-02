# TOTP Slack Command

[![Test](https://github.com/178inaba/totp-slack-command/actions/workflows/test.yml/badge.svg)](https://github.com/178inaba/totp-slack-command/actions/workflows/test.yml)

## Usage

```console
$ gcloud functions deploy GenerateTOTP \
    --gen2 \
    --trigger-http \
    --allow-unauthenticated \
    --runtime go122 \
    --env-vars-file .env.yaml \
    --region <region> \
    --project <project-id>
```

## License

[MIT](LICENSE)

## Author

Masahiro Furudate (a.k.a. [178inaba](https://github.com/178inaba))  
<178inaba.git@gmail.com>
