import React, { Component } from 'react';

import { Alert } from 'reactstrap';

export class ErrorMsg extends Component {
  constructor(props) {
    super(props)

    this.style = {
      color: 'red'
    };
  }

  render() {
    return (
      <Alert
        color="danger"
        style={this.style}
        isOpen={!!this.props.children}
      >
        {this.props.children ? 'Error: ' + this.props.children : ''}
      </Alert>
    )
  }
}

export default ErrorMsg;
