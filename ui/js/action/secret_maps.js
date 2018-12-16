import React, { Component } from 'react';
import { render } from 'react-dom';

import KVEntry from '../common/kv_entry';

class SecretMaps extends Component {
  constructor(props) {
    super(props);
  }

  render() {
    return (
      <KVEntry
        kvs={this.props.secretMaps}
        onChange={this.props.onChange}
        label="Add Vault Secret Paths/Mappings"
        placeholders={['Secret Ref', 'Vault Secret Path']}
      />
    );
  }
}

export default SecretMaps;
