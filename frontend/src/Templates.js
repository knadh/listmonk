import React from "react";
import {
  Row,
  Col,
  Modal,
  Form,
  Input,
  Button,
  Table,
  Icon,
  Tooltip,
  Tag,
  Popconfirm,
  Spin,
  notification
} from "antd";

import ModalPreview from "./ModalPreview";
import Utils from "./utils";
import * as cs from "./constants";

class CreateFormDef extends React.PureComponent {
  state = {
    confirmDirty: false,
    modalWaiting: false,
    previewName: "",
    previewBody: ""
  };

  // Handle create / edit form submission.
  handleSubmit = e => {
    e.preventDefault();
    this.props.form.validateFields((err, values) => {
      if (err) {
        return;
      }

      this.setState({ modalWaiting: true });
      if (this.props.formType === cs.FormCreate) {
        // Create a new list.
        this.props
          .modelRequest(
            cs.ModelTemplates,
            cs.Routes.CreateTemplate,
            cs.MethodPost,
            values
          )
          .then(() => {
            notification["success"]({
              placement: cs.MsgPosition,
              message: "Template added",
              description: `"${values["name"]}" added`
            });
            this.props.fetchRecords();
            this.props.onClose();
            this.setState({ modalWaiting: false });
          })
          .catch(e => {
            notification["error"]({
              placement: cs.MsgPosition,
              message: "Error",
              description: e.message
            });
            this.setState({ modalWaiting: false });
          });
      } else {
        // Edit a list.
        this.props
          .modelRequest(
            cs.ModelTemplates,
            cs.Routes.UpdateTemplate,
            cs.MethodPut,
            { ...values, id: this.props.record.id }
          )
          .then(() => {
            notification["success"]({
              placement: cs.MsgPosition,
              message: "Template updated",
              description: `"${values["name"]}" modified`
            });
            this.props.fetchRecords();
            this.props.onClose();
            this.setState({ modalWaiting: false });
          })
          .catch(e => {
            notification["error"]({
              placement: cs.MsgPosition,
              message: "Error",
              description: e.message
            });
            this.setState({ modalWaiting: false });
          });
      }
    });
  };

  handleConfirmBlur = e => {
    const value = e.target.value;
    this.setState({ confirmDirty: this.state.confirmDirty || !!value });
  };

  handlePreview = (name, body) => {
    this.setState({ previewName: name, previewBody: body });
  };

  render() {
    const { formType, record, onClose } = this.props;
    const { getFieldDecorator } = this.props.form;

    const formItemLayout = {
      labelCol: { xs: { span: 16 }, sm: { span: 4 } },
      wrapperCol: { xs: { span: 16 }, sm: { span: 18 } }
    };

    if (formType === null) {
      return null;
    }

    return (
      <div>
        <Modal
          visible={true}
          title={formType === cs.FormCreate ? "Add template" : record.name}
          okText={this.state.form === cs.FormCreate ? "Add" : "Save"}
          width="90%"
          height={900}
          confirmLoading={this.state.modalWaiting}
          onCancel={onClose}
          onOk={this.handleSubmit}
        >
          <Spin
            spinning={
              this.props.reqStates[cs.ModelTemplates] === cs.StatePending
            }
          >
            <Form onSubmit={this.handleSubmit}>
              <Form.Item {...formItemLayout} label="Name">
                {getFieldDecorator("name", {
                  initialValue: record.name,
                  rules: [{ required: true }]
                })(<Input autoFocus maxLength={200} />)}
              </Form.Item>
              <Form.Item {...formItemLayout} name="body" label="Raw HTML">
                {getFieldDecorator("body", {
                  initialValue: record.body ? record.body : "",
                  rules: [{ required: true }]
                })(<Input.TextArea autosize={{ minRows: 10, maxRows: 30 }} />)}
              </Form.Item>
              {this.props.form.getFieldValue("body") !== "" && (
                <Form.Item {...formItemLayout} colon={false} label="&nbsp;">
                  <Button
                    icon="search"
                    onClick={() =>
                      this.handlePreview(
                        this.props.form.getFieldValue("name"),
                        this.props.form.getFieldValue("body")
                      )
                    }
                  >
                    Preview
                  </Button>
                </Form.Item>
              )}
            </Form>
          </Spin>
          <Row>
            <Col span="4" />
            <Col span="18" className="text-grey text-small">
              The placeholder{" "}
              <code>
                {"{"}
                {"{"} template "content" . {"}"}
                {"}"}
              </code>{" "}
              should appear in the template.{" "}
              <a
                href="https://listmonk.app/docs/templating"
                target="_blank"
                rel="noopener noreferrer"
              >
                Learn more <Icon type="link" />.
              </a>
              .
            </Col>
          </Row>
        </Modal>

        {this.state.previewBody && (
          <ModalPreview
            title={
              this.state.previewName
                ? this.state.previewName
                : "Template preview"
            }
            previewURL={cs.Routes.PreviewNewTemplate}
            body={this.state.previewBody}
            onCancel={() => {
              this.setState({ previewBody: null, previewName: null });
            }}
          />
        )}
      </div>
    );
  }
}

const CreateForm = Form.create()(CreateFormDef);

class Templates extends React.PureComponent {
  state = {
    formType: null,
    record: {},
    previewRecord: null
  };

  constructor(props) {
    super(props);

    this.columns = [
      {
        title: "Name",
        dataIndex: "name",
        sorter: true,
        width: "50%",
        render: (text, record) => {
          return (
            <div className="name">
              <a role="button" onClick={() => this.handleShowEditForm(record)}>
                {text}
              </a>
              {record.is_default && (
                <div>
                  <Tag>Default</Tag>
                </div>
              )}
            </div>
          );
        }
      },
      {
        title: "Created",
        dataIndex: "created_at",
        render: (date, _) => {
          return Utils.DateString(date);
        }
      },
      {
        title: "Updated",
        dataIndex: "updated_at",
        render: (date, _) => {
          return Utils.DateString(date);
        }
      },
      {
        title: "",
        dataIndex: "actions",
        width: "20%",
        className: "actions",
        render: (text, record) => {
          return (
            <div className="actions">
              <Tooltip
                title="Preview template"
                onClick={() => this.handlePreview(record)}
              >
                <a role="button">
                  <Icon type="search" />
                </a>
              </Tooltip>

              {!record.is_default && (
                <Popconfirm
                  title="Are you sure?"
                  onConfirm={() => this.handleSetDefault(record)}
                >
                  <Tooltip title="Set as default" placement="bottom">
                    <a role="button">
                      <Icon type="check" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}

              <Tooltip title="Edit template">
                <a
                  role="button"
                  onClick={() => this.handleShowEditForm(record)}
                >
                  <Icon type="edit" />
                </a>
              </Tooltip>

              {record.id !== 1 && (
                <Popconfirm
                  title="Are you sure?"
                  onConfirm={() => this.handleDeleteRecord(record)}
                >
                  <Tooltip title="Delete template" placement="bottom">
                    <a role="button">
                      <Icon type="delete" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}
            </div>
          );
        }
      }
    ];
  }

  componentDidMount() {
    this.props.pageTitle("Templates");
    this.fetchRecords();
  }

  fetchRecords = () => {
    this.props.modelRequest(
      cs.ModelTemplates,
      cs.Routes.GetTemplates,
      cs.MethodGet
    );
  };

  handleDeleteRecord = record => {
    this.props
      .modelRequest(
        cs.ModelTemplates,
        cs.Routes.DeleteTemplate,
        cs.MethodDelete,
        { id: record.id }
      )
      .then(() => {
        notification["success"]({
          placement: cs.MsgPosition,
          message: "Template deleted",
          description: `"${record.name}" deleted`
        });

        // Reload the table.
        this.fetchRecords();
      })
      .catch(e => {
        notification["error"]({ message: "Error", description: e.message });
      });
  };

  handleSetDefault = record => {
    this.props
      .modelRequest(
        cs.ModelTemplates,
        cs.Routes.SetDefaultTemplate,
        cs.MethodPut,
        { id: record.id }
      )
      .then(() => {
        notification["success"]({
          placement: cs.MsgPosition,
          message: "Template updated",
          description: `"${record.name}" set as default`
        });

        // Reload the table.
        this.fetchRecords();
      })
      .catch(e => {
        notification["error"]({
          placement: cs.MsgPosition,
          message: "Error",
          description: e.message
        });
      });
  };

  handlePreview = record => {
    this.setState({ previewRecord: record });
  };

  hideForm = () => {
    this.setState({ formType: null });
  };

  handleShowCreateForm = () => {
    this.setState({ formType: cs.FormCreate, record: {} });
  };

  handleShowEditForm = record => {
    this.setState({ formType: cs.FormEdit, record: record });
  };

  render() {
    return (
      <section className="content templates">
        <Row>
          <Col xs={24} sm={14}>
            <h1>Templates ({this.props.data[cs.ModelTemplates].length}) </h1>
          </Col>
          <Col xs={24} sm={10} className="right header-action-break">
            <Button
              type="primary"
              icon="plus"
              onClick={this.handleShowCreateForm}
            >
              Add template
            </Button>
          </Col>
        </Row>
        <br />

        <Table
          columns={this.columns}
          rowKey={record => record.id}
          dataSource={this.props.data[cs.ModelTemplates]}
          loading={this.props.reqStates[cs.ModelTemplates] !== cs.StateDone}
          pagination={false}
        />

        <CreateForm
          {...this.props}
          formType={this.state.formType}
          record={this.state.record}
          onClose={this.hideForm}
          fetchRecords={this.fetchRecords}
        />

        {this.state.previewRecord && (
          <ModalPreview
            title={this.state.previewRecord.name}
            previewURL={cs.Routes.PreviewTemplate.replace(
              ":id",
              this.state.previewRecord.id
            )}
            onCancel={() => {
              this.setState({ previewRecord: null });
            }}
          />
        )}
      </section>
    );
  }
}

export default Templates;
