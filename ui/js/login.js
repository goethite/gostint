import React, { Component } from 'react';

import { InputGroup, Input, Button, Form } from 'reactstrap';

import { Poll } from "restful-react";

export class Login extends Component {
  constructor(props) {
    super(props);

    this.state = {
      // sessionFn: props.sessionFn,
      token: '',
      userName: '',
      userPassword: '',
      roleId: '',
      secretId: '',
      vaultAddr: ''
    };

    this.login = this.login.bind(this);
    this.handleChange = this.handleChange.bind(this);

    // get the vault url from gostint
    fetch(window.location.origin + '/v1/api/vault/info')
    .then(res => res.json())
    .then((res) => {
      console.log('GET ', res);
      this.setState(() => ({
        vaultAddr: res.vault_external_addr
      }));
    })
  }

  render() {
    return (
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
      </Form>
    );
  } // render

  handleChange(event) {
    console.log('handleChange event:', event);
    const target = event.target
    console.log('target:', target);
    console.log('target name:', target.name);
    console.log('target value:', target.value);

    if (target.name) {
      this.setState({
        [target.name]: target.value
      })
    }
  }

  login(event) {
    event.preventDefault();
    console.log('login clicked this:', this);
    console.log('event:', event);
    console.log('state:', this.state);

    if (this.state.token !== '') {
      fetch(this.state.vaultAddr + '/v1/auth/token/lookup-self', {
        headers: {
          'X-Vault-Token': this.state.token
        }
        // mode: 'no-cors'
      })
      .then(res => res.json())
      .then((res) => {
        console.log('GET self ', res);
      })
      .catch((err) => {
        console.error('vault lookup self err:', err);
      });

      // const xhr = new XMLHttpRequest();
      // xhr.open('GET', this.state.vaultAddr + '/v1/auth/token/lookup-self');
      // xhr.responseType = 'json';
      // xhr.setRequestHeader('X-Vault-Token', this.state.token);
      //
      // xhr.onload = function() {
      //   console.log(xhr.response);
      // };
      //
      // xhr.onerror = function(err) {
      //   console.log("Booo", err);
      // };
      //
      // xhr.send();

    }

    // if (this.state.sessionFn) {
    //   this.state.sessionFn(this.state.token);
    // }

  }
}
