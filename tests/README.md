
To create a base64 encoded targz for submitting in content:
```bash
tar zcf - . | base64 -w 0
```
