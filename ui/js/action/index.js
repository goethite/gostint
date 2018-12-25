import React, { Component } from 'react';
import { render } from 'react-dom';
import {
  Button,
  Col,
  Collapse,
  Container,
  Form,
  FormGroup,
  FormText,
  Input,
  InputGroup,
  Label,
  Row,
  Table
} from 'reactstrap';

import { default as AnsiUp } from 'ansi_up';
const ansi_up = new AnsiUp();

import ErrorMsg from '../error_message.js';
import EnvVars from './env_vars.js';
import SecretMaps from './secret_maps.js';
import Results from '../results';

import { gostint } from '../common/gostint_api.js';
import { vault } from '../common/vault_api.js';

import { parse } from 'shell-quote';

import css from './style.css';

class Action extends Component {
  constructor(props) {
    super(props);

    this.state = {
      actionShow: true,
      resultsShow: false,
      advancedForm: false,
      errorMessage: '',
      contentErrorMessage: '',

      dockerImage: '',
      run: '',
      imagePullPolicy: 'IfNotPresent',
      gostintRole: 'gostint-role',
      qName: '',
      content: '',
      entryPoint: '',
      workingDir: '',
      secretFileType: 'yaml',
      contOnWarnings: false,
      secretMaps: [],
      envVars: [],

      results: {},

      refreshResults: true
    };


    this.advancedSimple = this.advancedSimple.bind(this);
    this.run = this.run.bind(this);
    this.resultsReturn = this.resultsReturn.bind(this);
    this.viewResult = this.viewResult.bind(this);
    this.handleChange = this.handleChange.bind(this);
    this.handleSecretMaps = this.handleSecretMaps.bind(this);
    this.handleEnvVars = this.handleEnvVars.bind(this);
  }

  handleChange(event) {
    const target = event.target
    if (target.name) {
      switch(target.name) {
        case 'contOnWarnings':
          this.setState((state) => {
            return {'contOnWarnings': !state.contOnWarnings};
          });
          break;

        case 'content':
        const content = target.value
          this.setState({contentErrorMessage: ''}, () => {
            this.getBase64(target.files[0])
            .then((data) => {
              if (!data || data === '') { // Cancel pressed
                this.setState({
                  'content': '',
                  'contentB64': ''
                });
                return;
              }
              const parts = data.split(',');
              if (parts[0].search('gzip') === -1) {
                this.setState({
                  contentErrorMessage: 'not a gzip file',
                  'content': '',
                  'contentB64': ''
                });
                return;
              }
              this.setState((state) => {
                return {
                  'content': content,
                  'contentB64': 'targz,' + parts[1]
                };
              });
            });
          }); // setState contentErrorMessage
          break;

        default:
          this.setState({
            [target.name]: target.value
          });
      }
    }
  }

  render() {
    return (
      <div>
        <Collapse isOpen={this.state.actionShow}>
          <Form className={css.form} onSubmit={this.run} name="actionForm">
            <h4 className={css.hdr}>Action</h4>
            <Container>
              <br/>
              <Button
                color="secondary"
                className={css.advButton}
                onClick={this.advancedSimple}>{this.state.advancedForm ? 'Simple' : 'Advanced'}</Button>
              <br/>
              <Row>
                <Col md="3">
                  <FormGroup>
                    <Label for="dockerImage">Docker Image:</Label>
                    <Input
                      type="text"
                      name="dockerImage"
                      id="dockerImage"
                      placeholder="Enter docker image:tag"
                      onChange={this.handleChange}
                      value={this.state.dockerImage}
                    />
                  </FormGroup>
                </Col>

                <Col md="3">
                  <FormGroup>
                    <Label for="run">Container Run Command:</Label>
                    <Input
                      type="text"
                      name="run"
                      id="run"
                      placeholder="Enter A Run Command"
                      onChange={this.handleChange}
                      value={this.state.run}
                    />
                  </FormGroup>
                </Col>
              </Row>

              <Collapse isOpen={this.state.advancedForm}>
                <Row>
                  <Col md="3">
                    <FormGroup>
                      <Label for="imagePullPolicy">Image Pull Policy:</Label>
                      <Input
                        type="select"
                        name="imagePullPolicy"
                        id="imagePullPolicy"
                        value={this.state.imagePullPolicy}
                        placeholder="Enter Image Pull Policy: 'IfNotPresent' or 'Always'"
                        onChange={this.handleChange}
                      >
                        <option>IfNotPresent</option>
                        <option>Always</option>
                      </Input>
                    </FormGroup>
                  </Col>

                  <Col md="3">
                    <FormGroup>
                      <Label for="gostintRole">GoStint Vault AppRole Name:</Label>
                      <Input
                        type="text"
                        name="gostintRole"
                        id="gostintRole"
                        value={this.state.gostintRole}
                        placeholder="Enter GoStint Vault AppRole Name"
                        onChange={this.handleChange}
                      />
                    </FormGroup>
                  </Col>

                  <Col md="3">
                    <FormGroup>
                      <Label for="qName">Queue Name:</Label>
                      <Input
                        type="text"
                        name="qName"
                        id="qName"
                        placeholder="Enter A Queue Name (optional)"
                        onChange={this.handleChange}
                        value={this.state.qName}
                      />
                      <FormText>
                        Queues serialise jobs.  Jobs in different queues run in
                        parallel.
                      </FormText>
                    </FormGroup>
                  </Col>
                </Row>

                <FormGroup>
                  <Label for="content">Content:</Label>
                  <Input
                    type="file"
                    name="content"
                    id="content"
                    onChange={this.handleChange}
                    defaultValue={this.state.content}
                    accept="application/gzip"
                  />
                  <FormText>
                    Select a Gzipped Tar file of content to be injected into the container for
                    your job.
                  </FormText>
                  <ErrorMsg>{this.state.contentErrorMessage}</ErrorMsg>
                </FormGroup>

                <Row>
                  <Col md="3">
                    <FormGroup>
                      <Label for="entryPoint">Container Entry Point:</Label>
                      <Input
                        type="text"
                        name="entryPoint"
                        id="entryPoint"
                        placeholder="Enter Container Entry Point"
                        onChange={this.handleChange}
                        value={this.state.entryPoint}
                      />
                      <FormText>
                        Leave blank for default entrypoint (optional).
                      </FormText>
                    </FormGroup>
                  </Col>

                  <Col md="3">
                    <FormGroup>
                      <Label for="workingDir">Container Working Directory:</Label>
                      <Input
                        type="text"
                        name="workingDir"
                        id="workingDir"
                        placeholder="Enter Container Working Directory"
                        onChange={this.handleChange}
                        value={this.state.workingDir}
                      />
                      <FormText>
                        Leave blank for default.
                      </FormText>
                    </FormGroup>
                  </Col>

                  <Col md="3">
                    <FormGroup>
                      <Label for="secretFileType">Injected Secret File Type:</Label>
                      <Input
                        type="select"
                        name="secretFileType"
                        id="secretFileType"
                        value={this.state.secretFileType}
                        onChange={this.handleChange}
                        >
                          <option>yaml</option>
                          <option>json</option>
                        </Input>
                    </FormGroup>
                  </Col>
                </Row>

                <Row>
                  <Col>
                    <FormGroup check inline>
                      <Label check>
                        <Input
                          type="checkbox"
                          name="contOnWarnings"
                          id="contOnWarnings"
                          onChange={this.handleChange}
                          checked={this.state.contOnWarnings}
                          value="toggle"
                        />
                        Continue on Warnings
                      </Label>
                    </FormGroup>
                    <FormText>
                    Continue to run job even if vault reported warnings when looking up secret refs.
                    </FormText>
                  </Col>
                </Row>

                <SecretMaps secretMaps={this.state.secretMaps} onChange={this.handleSecretMaps} />
                <EnvVars envVars={this.state.envVars} onChange={this.handleEnvVars} />

              </Collapse>

              <br/>
              <Button color="primary" className={css.runButton} type="submit">Run</Button>
              <br/>
              <ErrorMsg>{this.state.errorMessage}</ErrorMsg>
            </Container>
          </Form>

          <Results
            URLs={this.props.URLs}
            vaultAuth={this.props.vaultAuth}
            refresh={this.state.refreshResults}
            resultCb={this.viewResult}/>
        </Collapse>

        <Collapse isOpen={this.state.resultsShow}>
          <br/>
          <Button
            color="primary"
            className={css.returnButton}
            onClick={this.resultsReturn}>&lt;&lt; Return</Button>
          <br/>
          <Table>
            <thead>
              <tr>
                <th>Queue</th>
                <th>Status</th>
                <th>Started</th>
                <th>Ended</th>
                <th>Return Code</th>
              </tr>
            </thead>
            <tbody>
              <tr className={css._id}>
                <td>{this.state.results.qname}</td>
                <td>{this.state.results.status}</td>
                <td>{this.state.results.started}</td>
                <td>{
                  (this.state.results.status && this.state.results.status.match(/(running|queued)/)) ? '' : this.state.results.ended
                }</td>
                <td>{
                  (this.state.results.status && this.state.results.status.match(/(running|queued)/)) ? '' : this.state.results.return_code
                }</td>
              </tr>
            </tbody>
          </Table>

          <h3 className={css.outputHdr}>Output:</h3>
          {this.state.results.output ?
            <pre className={css.output}
              dangerouslySetInnerHTML={{
              __html: ansi_up.ansi_to_html(this.state.results.output)
            }}></pre>
          :
            <h3 className={css.outputElips}>...</h3>
          }
          <ErrorMsg>{this.state.errorMessage}</ErrorMsg>

        </Collapse>
      </div>
    );
  }

  resultsReturn() {
    this.setState(() => {
      return {
        actionShow: true,
        resultsShow: false,
        refreshResults: !this.state.refreshResults // refresh results table
      };
    });
  }

  viewResult(id) {
    console.log('viewResult id:', id);

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
        `v1/api/job/${id}`,
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
      this.setState(() => {
        return {
          errorMessage: '',
          actionShow: false,
          resultsShow: true,
          results: res
        };
      })
      // this.setState({results: res});
    })
    .catch((err) => {
      console.error('results err:', err);
    });
  }

  getBase64(file) {
    if (!file) {
      return Promise.resolve('');
    }
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = () => resolve(reader.result);
      reader.onerror = error => reject(error);
    });
  }

  run() {
    event.preventDefault();

    this.setState(() => {
      return {
        errorMessage: '',
        actionShow: false,
        resultsShow: true,
        results: {}
      };
    }, () => {
      // see https://github.com/goethite/gostint-client/blob/master/clientapi/clientapi.go
      const job = this.buildJob()
      let apiToken;
      let wrapSecretID;
      let encryptedJob;
      let cubbyToken;

      // Get a minimal token for job submission to gostint
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

        // Get secret id for gostint approle
        return vault(
          this.props.URLs.vault,
          this.props.vaultAuth.token,
          `v1/auth/approle/role/${this.state.gostintRole}/secret-id`,
          'POST'
        );
      })
      .then((res) => {
        // Wrap the secret id
        return vault(
          this.props.URLs.vault,
          this.props.vaultAuth.token,
          'v1/sys/wrapping/wrap',
          'POST',
          res.data,
          {'X-Vault-Wrap-TTL': 300}
        );
      })
      .then((res) => {
        wrapSecretID = res.wrap_info.token;

        // Encrypt the job payload
        const jobB64 = Buffer.from(JSON.stringify(job)).toString('base64');
        return vault(
          this.props.URLs.vault,
          this.props.vaultAuth.token,
          `v1/transit/encrypt/${this.state.gostintRole}`,
          'POST',
          {plaintext: jobB64}
        );
      })
      .then((res) => {
        encryptedJob = res.data.ciphertext;

        // get limited use token for cubbyhole
        return vault(
          this.props.URLs.vault,
          this.props.vaultAuth.token,
          'v1/auth/token/create',
          'POST',
          {
            policies: ['default'],
            ttl: '1h',
            use_limit: 2,
            display_name: 'gostint_cubbyhole'
          }
        );
      })
      .then((res) => {
        cubbyToken = res.auth.client_token;

        // Put encrypted job in cubbyhole
        return vault(
          this.props.URLs.vault,
          this.props.vaultAuth.token,
          'v1/cubbyhole/job',
          'POST',
          {payload: encryptedJob},
          {'X-Vault-Token': cubbyToken}
        );
      })
      .then(() => {
        // create job wrapper
        const jWrap = {
          qname: this.state.qName,
          cubby_token: cubbyToken,
          cubby_path: 'cubbyhole/job',
          wrap_secret_id: wrapSecretID
        };

        // submit job
        return gostint(
          this.props.URLs.gostint,
          'v1/api/job',
          'POST',
          jWrap,
          {
            'X-Auth-Token': apiToken,
            'Content-Type': 'application/json'
          }
        );
      })
      .then((res) => {
        return res.json()
      })
      .then((data) => {
        (function (self, apiToken, data) {
          const intvl = setInterval(() => {
            gostint(
              self.props.URLs.gostint,
              `v1/api/job/${data._id}`,
              'GET',
              null,
              {
                'X-Auth-Token': apiToken,
                'Content-Type': 'application/json'
              }
            )
            .then((queueRes) => {
              return queueRes.json();
            })
            .then((queue) => {
              self.setState(() => {
                return {
                  results: queue
                };
              });

              if (queue.status !== 'queued' && queue.status !== 'running' ) {
                clearInterval(intvl);
              }
            })
            .catch((err) => {
              console.error('err:', err);
              this.setState({errorMessage: err.message});
            });
          }, 2000);
        })(this, apiToken, data);

      })
      .catch((err) => {
        console.error('err:', err);
        this.setState({errorMessage: err.message});
      })
    });
  }

  buildJob() {
    return {
      qname:              this.state.qName,
      container_image:    this.state.dockerImage,
      image_pull_policy:  this.state.imagePullPolicy,
      content:            this.state.contentB64 || '',
      entrypoint:         parse(this.state.entryPoint),
      run:                parse(this.state.run),
      working_directory:  this.state.workingDir,
      env_vars:           this.state.envVars.map(er => `${er.key}=${er.val}`),
      secret_refs:        this.state.secretMaps.map(sr => `${sr.key}@${sr.val}`),
      secret_file_type:   this.state.secretFileType,
      cont_on_warnings:   this.state.contOnWarnings
    };
  }

  advancedSimple() {
    this.setState((state, props) => {
      return {advancedForm: !state.advancedForm};
    });
  }

  handleSecretMaps(action, row, idx) {
    switch(action) {
      case 'add':
        this.setState((state, props) => {
          const sr = Object.assign([], state.secretMaps);
          sr.push(row);
          return {secretMaps: sr};
        });
        break;
      case 'delete':
        this.setState((state) => {
          const sr = Object.assign([], state.secretMaps);
          sr.splice(idx, 1);
          return { secretMaps: sr };
        });
        break;
      default:
        console.error('handleSecretMaps unrecognised action:', action);
    }
  }

  handleEnvVars(action, row, idx) {
    switch(action) {
      case 'add':
        this.setState((state, props) => {
          const es = Object.assign([], state.envVars);
          es.push(row);
          return {envVars: es};
        });
        break;
      case 'delete':
        this.setState((state) => {
          const es = Object.assign([], state.envVars);
          es.splice(idx, 1);
          return { envVars: es };
        });
        break;
      default:
        console.error('handleEnvVars unrecognised action:', action);
    }
  }
}

export default Action;
