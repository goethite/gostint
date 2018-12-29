import React, { Component } from 'react';

import { InputGroup, Input, Button, Form } from 'reactstrap';
// import { Get } from "restful-react";
import { withRouter } from 'react-router-dom';

import ErrorMsg from './error_message.js';
import { vault } from './common/vault_api.js';

const rootPath = window.location.pathname === '/' ? '' : window.location.pathname;

export class Login extends Component {
  constructor(props) {
    super(props);

    let { from } = props.location.state || { from: { pathname: "/" } };

    this.state = {
      sessionFn: props.sessionFn,
      from,
      vaultAuth: props.vaultAuth,
      token: '',
      gostintToken: '',
      userName: '',
      userPassword: '',
      roleId: '',
      secretId: '',
      vaultAddr: '',
      errorMessage: ''
    };

    this.login = this.login.bind(this);
    this.handleChange = this.handleChange.bind(this);

    // get the vault url from gostint
    fetch(`${window.location.origin}${rootPath}/v1/api/vault/info`)
    .then((res) => {
      if (res.status !== 200) {
        return Promise.reject(
          new Error(`Request for vault info from gostint failed with status: ${res.status} ${res.statusText} [${res.url}]`)
        );
      }
      return res.json();
    })
    .then((res) => {
      this.setState(() => ({
        vaultAddr: res.vault_external_addr
      }));
    })
    .catch((err) => {
      console.error('get vault info err:', err);
      this.setState(() => ({
        errorMessage: err.message,
        vaultAddr: 'INVALID'
      }));
    });
  }

  render() {
    return (
      <div>
        <h1 className="siteHeader">GoStint</h1>
        <Form className="loginContainer" onSubmit={this.login}>
          <h4 className="loginHdr">Login</h4>

          <label className="loginLabel">with a Vault Token:</label>
          <InputGroup className="loginInput">
            <Input
              type="password"
              name="token"
              placeholder="Token"
              onChange={this.handleChange} />
          </InputGroup>

          {/* TODO:
          <label className="loginLabel">or Username and Password:</label>
          <InputGroup className="loginInput">
            <Input name="userName" placeholder="User name" onChange={this.handleChange} />
          </InputGroup>
          <br/>
          <InputGroup className="loginInput">
            <Input name="userPassword" placeholder="Password" onChange={this.handleChange} />
          </InputGroup>

          <label className="loginLabel">or an AppRole:</label>
          <InputGroup className="loginInput">
            <Input name="roleId" placeholder="AppRole ID" onChange={this.handleChange} />
          </InputGroup>
          <br/>
          <InputGroup className="loginInput">
            <Input name="secretId" placeholder="Secret ID" onChange={this.handleChange} />
          </InputGroup>
          */}

          <br/>
          <Button color="primary" className="loginButton" type="submit">Login</Button>
          <br/>
          <ErrorMsg>{this.state.errorMessage}</ErrorMsg>
        </Form>
      </div>
    );
  } // render

  handleChange(event) {
    const target = event.target
    if (target.name) {
      this.setState({
        [target.name]: target.value
      })
    }
  }

  login(event) {
    event.preventDefault();

    this.setState(() => ({errorMessage: ''}));

    if (this.state.vaultAddr === '' || this.state.vaultAddr === 'INVALID') {
      this.setState(() => ({errorMessage: 'Invalid Vault address'}));
      return;
    }

    if (this.state.token !== '') {
      const vaultURL = this.state.vaultAddr + '/v1/auth/token/lookup-self';
      fetch(vaultURL, {
        headers: {
          'X-Vault-Token': this.state.token
        }
        // mode: 'no-cors'
      })
      .then((res) => {
        if (res.status !== 200) {
          return Promise.reject(
            new Error(`Request failed with status: ${res.status} ${res.statusText} [${res.url}]`)
          );
        }
        return res.json();
      })
      .then((res) => {
        if (res.errors) {
          // this.setState(() => ({errorMessage: res.errors.join(', ')}));
          return Promise.reject(new Error(res.errors.join(', ')));
        }

        // Get a minimal token for gostint api, this is so we dont handover
        // the requestor's actual cred/permissions to gostint (e.g. the ability
        // to request the approle's secret_id that authorizes gostint to access
        // secrets in the vault).
        return vault(
          this.state.vaultAddr,
          this.state.token,
          'v1/auth/token/create',
          'POST',
          {
            policies: ['default'],
            ttl: '6h',
            // ttl: '30s',
            // num_uses: 1,
            display_name: 'gostint_ui'
          }
        )
      })
      .then((res) => {
        this.state.gostintToken = res.auth.client_token;

        if (this.state.sessionFn) {
          this.state.sessionFn({
            token: this.state.token,
            gostintToken: this.state.gostintToken,
            // from: this.state.from,
            originURL: window.location.origin,
            vaultURL: this.state.vaultAddr
          });
        }
        this.props.history.push("/");
      })
      .catch((err) => {
        console.error('vault token err:', err);
        this.setState(() => ({errorMessage: err.message}));
      });
    } // if token
  }
}

export default withRouter(Login);
