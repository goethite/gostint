import React, { Component } from 'react';
import { render } from 'react-dom';
import {
  Button,
  Col,
  FormGroup,
  Input,
  Label,
  Row,
  Table
} from 'reactstrap';

import { FaTrashAlt } from 'react-icons/fa';

import css from './style.css';

class KVEntry extends Component {
  constructor(props) {
    super(props);

    this.state = {
      key: '',
      val: '',
      onChange: props.onChange || null,
      label: props.label || 'Enter Keys/Values',
      placeholders: props.placeholders || ['key', 'value']
    };

    this.addKV = this.addKV.bind(this);
    this.deleteKV = this.deleteKV.bind(this);
    this.handleChange = this.handleChange.bind(this);
  }

  handleChange(event) {
    const target = event.target
    if (target.name) {
      this.setState({
        [target.name]: target.value
      })
    }
  }

  render() {
    return (
      <div className={css.container}>
        <Row className={css.rowEnvsSecs}>
          <Col><Label>{this.state.label}:</Label></Col>
        </Row>

        <Row>
          <Col md="3">
            <FormGroup>
              <Input
                type="text"
                name="key"
                id="key"
                placeholder={'Enter ' + this.state.placeholders[0]}
                onChange={this.handleChange}
                value={this.state.key}
              />
            </FormGroup>
          </Col>
          <Col md="9">
            <FormGroup>
              <Input
                type="text"
                name="val"
                id="val"
                placeholder={'Enter ' + this.state.placeholders[1]}
                onChange={this.handleChange}
                value={this.state.val}
              />
            </FormGroup>
          </Col>
        </Row>

        <Row>
          <Col>
            <Button
              color="secondary"
              className={css.addButton}
              disabled={!this.state.key || !this.state.val}
              onClick={this.addKV}>Add</Button>
            <Table striped size="sm">
              <thead>
                <tr className="d-flex">
                  <th className="col-md-3">{this.state.placeholders[0]}</th>
                  <th className="col-md-9">{this.state.placeholders[1]}</th>
                </tr>
              </thead>
              <tbody>
              {this.props.kvs.map((r, i) => {
                return (
                  <tr
                    className="d-flex"
                    key={i.toString()}>
                    <td className="col-md-3">{r.key}</td>
                    <td className="col-md-9">
                      {r.val}
                      <Button color="danger"
                        className={css.deleteButton}
                        data-item={i}
                        onClick={this.deleteKV}><FaTrashAlt /></Button>
                    </td>
                  </tr>
                );
              })}
              </tbody>
            </Table>
          </Col>
        </Row>
      </div>
    );
  }

  addKV() {
    this.state.onChange('add', {
      key: this.state.key,
      val: this.state.val
    });

    // reset input fields
    this.setState((state) => {
      return {
        key: '',
        val: ''
      };
    });
  }

  deleteKV(e) {
    const idx = e.currentTarget.getAttribute('data-item');
    this.state.onChange('delete', {}, idx);
  }
}

export default KVEntry;
