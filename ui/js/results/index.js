import React, { Component } from 'react';
import { render } from 'react-dom';
import {
  Button,
  Pagination,
  PaginationItem,
  PaginationLink,
  Table
} from 'reactstrap';

import { gostint } from '../common/gostint_api.js';
import { vault } from '../common/vault_api.js';

import css from './style.css';

class Results extends Component {
  constructor(props) {
    super(props);

    this.state = {
      skip: 0,
      results: {}
    }

    this.intvl;

    this.selectResult = this.selectResult.bind(this);
    this.resultsPrevious = this.resultsPrevious.bind(this);
    this.resultsNext = this.resultsNext.bind(this);
  }

  componentDidMount() {
    console.log('in componentDidMount');
    this.refreshResults();

    this.intvl = setInterval(() => {
      this.refreshResults();
    }, 10000);
  }

  componentWillUnmount() {
    clearInterval(this.intvl);
  }

  componentWillReceiveProps(props) {
    console.log('in componentWillReceiveProps props:', props);
    const { refresh } = this.props;
    if (props.refresh !== refresh) {
      this.refreshResults();
    }
  }

  refreshResults() {
    let apiToken;
    // Get a minimal token for job query to gostint
    vault(
      this.props.URLs.vault,
      this.props.vaultAuth.token,
      'v1/auth/token/create',
      'POST',
      {
        policies: ['default'],
        ttl: '6h',
        // num_uses: 1,
        display_name: 'gostint_ui'
      }
    )
    .then((res) => {
      apiToken = res.auth.client_token;

      return gostint(
        this.props.URLs.gostint,
        `v1/api/job?skip=${this.state.skip}`,
        'GET',
        null,
        {
          'X-Auth-Token': apiToken,
          'Content-Type': 'application/json'
        }
      )
    })
    .then((res) => res.json())
    .then((res) => {
      console.log('results res:', res);
      this.setState({results: res});
    })
    .catch((err) => {
      console.error('results err:', err);
    });
  }

  selectResult(e) {
    console.log('selectResult clicked e:', e);
    const resultId = e.currentTarget.getAttribute('data-item');
    console.log('resultId:', resultId)
    this.props.resultCb(resultId);
  }

  resultsPrevious() {
    console.log('resultsPrevious');
    this.setState((state) => {
      return {
        skip: state.skip - 10 > 0 ? state.skip - 10 : 0
      };
    }, () => {
      this.refreshResults();
    });
  }

  resultsNext() {
    console.log('resultsNext');
    this.setState((state) => {
      return {
        skip: state.skip + 10 < this.state.results.total ? state.skip + 10 : state.skip
      };
    }, () => {
      this.refreshResults();
    });
  }

  render() {
    return (
      <div className={css.container}>
        <h4 className={css.hdr}>Results</h4>

        <Table striped size="sm">
          <thead>
            <tr>
              <th>ID</th>
              <th>Queue</th>
              <th>Status</th>
              <th>Image</th>
              <th>Submitted</th>
              <th>Started</th>
              <th>Ended</th>
              <th>Return Code</th>
            </tr>
          </thead>
          <tbody>
            {this.state.results.data && this.state.results.data.map((r, i) => {
              return (
                <tr
                  className={css.row}
                  key={i.toString()}
                  data-item={r._id}
                  onClick={this.selectResult}>
                  <td>{r._id}</td>
                  <td>{r.qname}</td>
                  <td>{r.status}</td>
                  <td>{r.container_image}</td>
                  <td>{r.submitted}</td>
                  <td>{r.started}</td>
                  <td>{r.ended}</td>
                  <td>{r.return_code}</td>
                </tr>
              );
            })}
          </tbody>
        </Table>

        <div className="text-center">
          <Pagination
            className={css.paginationCenter}
            aria-label="Paginate Results">
            <PaginationItem>
              <PaginationLink
                previous
                onClick={this.resultsPrevious}
              />
            </PaginationItem>

            <PaginationItem>
              <PaginationLink
                next
                onClick={this.resultsNext}
              />
            </PaginationItem>
          </Pagination>
        </div>
      </div>
    );
  }
}

export default Results;
