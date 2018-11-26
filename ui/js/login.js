const Component = React.Component;

export class Login extends Component {
  constructor(props) {
    super(props);

    this.state = {
      email: "",
      password: ""
    };
  }

  render() {
    return (
      React.createElement('div', { className: 'loginContainer' },
        React.createElement('h4', { className: 'loginHdr' }, 'Login'),
        React.createElement('label', { className: 'loginLabel' }, 'using a Vault Token:'),
        React.createElement(Reactstrap.InputGroup, { className: 'loginInput' },
          // React.createElement(Reactstrap.InputGroupAddon, {addonType: 'prepend'},
          //   React.createElement(Reactstrap.InputGroupText, {}, 'Token')
          // ),
          React.createElement(Reactstrap.Input, {
            placeholder: 'Token'
          })
        ),
        React.createElement('label', { className: 'loginLabel' }, 'or Username and Password:'),
        React.createElement(Reactstrap.InputGroup, { className: 'loginInput' },
          // React.createElement(Reactstrap.InputGroupAddon, {addonType: 'prepend'},
          //   React.createElement(Reactstrap.InputGroupText, {}, 'User name')
          // ),
          React.createElement(Reactstrap.Input, {
            placeholder: 'User name'
          }),
        ),
        React.createElement('br', {}),
        React.createElement(Reactstrap.InputGroup, { className: 'loginInput' },
          // React.createElement(Reactstrap.InputGroupAddon, {addonType: 'prepend'},
          //   React.createElement(Reactstrap.InputGroupText, {}, 'Password')
          // ),
          React.createElement(Reactstrap.Input, {
            placeholder: 'Password'
          })
        ),
        React.createElement('label', { className: 'loginLabel' }, 'or an AppRole:'),
        React.createElement(Reactstrap.InputGroup, { className: 'loginInput' },
          // React.createElement(Reactstrap.InputGroupAddon, {addonType: 'prepend'},
          //   React.createElement(Reactstrap.InputGroupText, {}, 'AppRole ID')
          // ),
          React.createElement(Reactstrap.Input, {
            placeholder: 'AppRole ID'
          }),
        ),
        React.createElement('br', {}),
        React.createElement(Reactstrap.InputGroup, { className: 'loginInput' },
          // React.createElement(Reactstrap.InputGroupAddon, {addonType: 'prepend'},
          //   React.createElement(Reactstrap.InputGroupText, {}, 'Secret ID')
          // ),
          React.createElement(Reactstrap.Input, {
            placeholder: 'Secret ID'
          })
        ),
        React.createElement('br', {}),
        React.createElement(
          Reactstrap.Button,
          { color: 'primary', className: 'loginButton' },
          'Login'
        ),
      )
    );
  }
}
