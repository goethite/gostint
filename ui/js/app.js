import { Login } from '/js/login.js';

const node = document.getElementById('gostint');
ReactDOM.render(
  React.createElement('div', {},
    React.createElement('h1', {}, 'gostint ui'),
    React.createElement(Login, {})
  ),
  node
);
