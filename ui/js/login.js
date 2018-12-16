import React, { Component } from 'react';

import { InputGroup, Input, Button, Form } from 'reactstrap';
// import { Get } from "restful-react";
import { withRouter } from 'react-router-dom';

import ErrorMsg from './error_message.js';

export class Login extends Component {
  constructor(props) {
    super(props);
    console.log('props:', props);

    let { from } = props.location.state || { from: { pathname: "/" } };

    this.state = {
      sessionFn: props.sessionFn,
      from,
      vaultAuth: props.vaultAuth,
      token: '',
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
    fetch(window.location.origin + '/v1/api/vault/info')
    .then((res) => {
      if (res.status !== 200) {
        return Promise.reject(
          new Error(`Request for vault info from gostint failed with status: ${res.status} ${res.statusText} [${res.url}]`)
        );
      }
      return res.json();
    })
    .then((res) => {
      console.log('GET vault info from gostint', res);
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
        <h1 className="siteHeader">gostint</h1>
        <Form className="loginContainer" onSubmit={this.login}>
          <h4 className="loginHdr">Login</h4>

          <label className="loginLabel">with a Vault Token:</label>
          <InputGroup className="loginInput">
            <Input name="token" placeholder="Token" onChange={this.handleChange} />
          </InputGroup>

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
    console.log('event:', event);
    console.log('state:', this.state);

    this.setState(() => ({errorMessage: ''}));

    if (this.state.vaultAddr === '' || this.state.vaultAddr === 'INVALID') {
      this.setState(() => ({errorMessage: 'Invalid Vault address'}));
      return;
    }

    if (this.state.token !== '') {
      const vaultURL = this.state.vaultAddr + '/v1/auth/token/lookup-self';
      console.log('fetching', vaultURL);
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
        console.log('GET token self', res);
        if (res.errors) {
          console.log('in error');
          this.setState(() => ({errorMessage: res.errors.join(', ')}));
          console.log('this.state.errorMessage:', this.state.errorMessage);
        } else {
          console.log('logged in this.state.sessionFn:', this.state.sessionFn);
          if (this.state.sessionFn) {
            this.state.sessionFn({
              tokenData: this.state.token,
              // from: this.state.from,
              originURL: window.location.origin,
              vaultURL: this.state.vaultAddr
            });
          }
          console.log("login props:", this.props)
          this.props.history.push("/");
        }
      })
      .catch((err) => {
        console.error('vault lookup self err:', err);
        this.setState(() => ({errorMessage: err.message}));
      });
    } // if token
  }
}

export default withRouter(Login);
