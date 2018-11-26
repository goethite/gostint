
const node = document.getElementById('gostint');
console.log('node:', node);

const gostint =
  React.createElement('div', {},
    React.createElement('h1', {}, 'GoStint'),
    React.createElement('div', {}, 'Actually the app goes here...')
  );
ReactDOM.render(gostint, node);
