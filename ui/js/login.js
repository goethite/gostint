import React, { Component } from 'react';

import { InputGroup, Input, Button } from 'reactstrap';

export class Login extends Component {
  constructor(props) {
    super(props);

    this.state = {
      sessionFn: props.sessionFn,
      token: '',
      userName: '',
      userPassword: '',
      roleId: '',
      secretId: ''
    };

    this.login = this.login.bind(this);
    this.handleChange = this.handleChange.bind(this);
  }

  render() {
    return (
      <form className="loginContainer" onSubmit={this.login}>
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
      </form>
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
    console.log('login clicked this:', this);
    console.log('event:', event);
    console.log('state:', this.state);

    if (this.state.sessionFn) {
      this.state.sessionFn(this.state.token);
    }
    event.preventDefault();
  }
}
