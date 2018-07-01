# goswim

## Notes

### Testing Ephemeral user/password for MongoDB
`vagrant ssh` into the container
```
~$ vault login root
Success! You are now authenticated. The token information displayed below
is already stored in the token helper. You do NOT need to run "vault login"
again. Future Vault requests will automatically use this token.

Key                  Value
---                  -----
token                root
token_accessor       0a4e9bad-768b-3f2d-be35-afdb0b6f35c1
token_duration       ∞
token_renewable      false
token_policies       ["root"]
identity_policies    []
policies             ["root"]

~$ vault read database/creds/goswim-dbauth-role
Key                Value
---                -----
lease_id           database/creds/goswim-dbauth-role/9f12e958-a2e7-080e-e9df-b8842cb3f8ae
lease_duration     1h
lease_renewable    true
password           A1a-4bHwB9x6vd6irH51
username           v-token-goswim-dbauth-role-g0YkRCwmxnbnTcFh0oQ8-1530388299
