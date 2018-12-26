import React from 'react';
import { render } from 'react-dom';

import {
  BrowserRouter as Router,
  Route,
  Redirect,
  withRouter
} from 'react-router-dom';

import App from './app.js';
import Login from './login.js';

const vaultAuth = {
  token: null,        // requestor's vault token
  gostintToken: null  // minimal token for gotstint api access
}

const URLs = {
  gostint: '',
  vault: ''
}

const node = document.getElementById('gostint');
render(
  <Router>
    <div>
      <Route
        path="/login"
        render={props => <Login vaultAuth={vaultAuth} location={props.location} sessionFn={handleLogin} />}
      />
      <Route
        exact path="/"
        render={(props) => {
          props.URLs = URLs;
          props.vaultAuth = vaultAuth;
          return vaultAuth.token ? (
            <App {...props} />
          ) : (
            <Redirect
              to={{
                pathname: "/login",
                state: {from: props.location}
              }}
            />
          )
        }}
      />
    </div>
  </Router>
  , node
);

function handleLogin(data) {
  vaultAuth.token = data.token;
  vaultAuth.gostintToken = data.gostintToken;
  URLs.gostint = data.originURL;
  URLs.vault = data.vaultURL;
}
