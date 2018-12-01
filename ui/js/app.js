import { Login } from './login.js';
import React from 'react';
import { render } from 'react-dom';

const node = document.getElementById('gostint');
render(
    <div>
      <h1 className="siteHeader">gostint</h1>
      <Login/>
    </div>
  , node
);
