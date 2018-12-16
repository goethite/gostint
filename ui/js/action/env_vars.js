import React, { Component } from 'react';
import { render } from 'react-dom';

import KVEntry from '../common/kv_entry';

class EnvVars extends Component {
  constructor(props) {
    super(props);
  }

  render() {
    return (
      <KVEntry
        kvs={this.props.envVars}
        onChange={this.props.onChange}
        label="Container Environment Variables"
        placeholders={['Name', 'Value']}
      />
    );
  }
}

export default EnvVars;
