import React, { Component } from 'react';
import { render } from 'react-dom';
import {
  Button,
  Form,
  FormGroup,
  Input,
  InputGroup,
  Label
} from 'reactstrap';

import ToolBar from './toolbar';
import Action from './action';
import ErrorMsg from './error_message.js';

class App extends Component {
  constructor(props) {
    super(props);

    this.state = {
      token: (this.props ? this.props.vaultAuth.token : null)
    };
  }

  render() {
    return (
      <div>
        <ToolBar/>

        <Action URLs={this.props.URLs} vaultAuth={this.props.vaultAuth} />

      </div>
    );
  }
}

export default App;
