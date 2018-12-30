import React, { Component } from 'react';
import { render } from 'react-dom';

import {
  Navbar,
  NavbarBrand,
  Nav,
  NavItem,
  NavLink
} from 'reactstrap';

const rootPath = window.location.pathname === '/' ? '' : window.location.pathname;

class ToolBar extends Component {
  constructor() {
    super();
    this.state = {
      token: null
    };
  }

  render() {
    return (
      <div>
        <Navbar color="light" light expand="md">
          <NavbarBrand>GoStint</NavbarBrand>
          <Nav className="ml-auto" navbar>
            <NavItem>
              <NavLink href={rootPath}>Logout</NavLink>
            </NavItem>
          </Nav>
        </Navbar>
      </div>
    );
  }
}

export default ToolBar;
