import React from "react"
import { Link } from "react-router-dom"
import {
  Row,
  Col,
  Modal,
  Form,
  Input,
  Select,
  Button,
  Table,
  Icon,
  Tooltip,
  Tag,
  Popconfirm,
  Spin,
  notification
} from "antd"

import Utils from "./utils"
import * as cs from "./constants"

const tagColors = {
  private: "orange",
  public: "green"
}

class CreateFormDef extends React.PureComponent {
  state = {
    confirmDirty: false,
    modalWaiting: false
  }

  // Handle create / edit form submission.
  handleSubmit = e => {
    e.preventDefault()
    this.props.form.validateFields((err, values) => {
      if (err) {
        return
      }

      this.setState({ modalWaiting: true })
      if (this.props.formType === cs.FormCreate) {
        // Create a new list.
        this.props
          .modelRequest(
            cs.ModelLists,
            cs.Routes.CreateList,
            cs.MethodPost,
            values
          )
          .then(() => {
            notification["success"]({
              placement: cs.MsgPosition,
              message: "List created",
              description: `"${values["name"]}" created`
            })
            this.props.fetchRecords()
            this.props.onClose()
            this.setState({ modalWaiting: false })
          })
          .catch(e => {
            notification["error"]({ message: "Error", description: e.message })
            this.setState({ modalWaiting: false })
          })
      } else {
        // Edit a list.
        this.props
          .modelRequest(cs.ModelLists, cs.Routes.UpdateList, cs.MethodPut, {
            ...values,
            id: this.props.record.id
          })
          .then(() => {
            notification["success"]({
              placement: cs.MsgPosition,
              message: "List modified",
              description: `"${values["name"]}" modified`
            })
            this.props.fetchRecords()
            this.props.onClose()
            this.setState({ modalWaiting: false })
          })
          .catch(e => {
            notification["error"]({
              placement: cs.MsgPosition,
              message: "Error",
              description: e.message
            })
            this.setState({ modalWaiting: false })
          })
      }
    })
  }

  modalTitle(formType, record) {
    if (formType === cs.FormCreate) {
      return "Create a list"
    }

    return (
      <div>
        <Tag
          color={
            tagColors.hasOwnProperty(record.type) ? tagColors[record.type] : ""
          }
        >
          {record.type}
        </Tag>{" "}
        {record.name}
        <br />
        <span className="text-tiny text-grey">
          ID {record.id} / UUID {record.uuid}
        </span>
      </div>
    )
  }

  render() {
    const { formType, record, onClose } = this.props
    const { getFieldDecorator } = this.props.form

    const formItemLayout = {
      labelCol: { xs: { span: 16 }, sm: { span: 4 } },
      wrapperCol: { xs: { span: 16 }, sm: { span: 18 } }
    }

    if (formType === null) {
      return null
    }

    return (
      <Modal
        visible={true}
        title={this.modalTitle(this.state.form, record)}
        okText={this.state.form === cs.FormCreate ? "Create" : "Save"}
        confirmLoading={this.state.modalWaiting}
        onCancel={onClose}
        onOk={this.handleSubmit}
      >
        <div id="modal-alert-container" />

        <Spin
          spinning={this.props.reqStates[cs.ModelLists] === cs.StatePending}
        >
          <Form onSubmit={this.handleSubmit}>
            <Form.Item {...formItemLayout} label="Name">
              {getFieldDecorator("name", {
                initialValue: record.name,
                rules: [{ required: true }]
              })(<Input autoFocus maxLength={200} />)}
            </Form.Item>
            <Form.Item
              {...formItemLayout}
              name="type"
              label="Type"
              extra="Public lists are open to the world to subscribe and their
              names may appear on public pages such as the subscription management page."
            >
              {getFieldDecorator("type", {
                initialValue: record.type ? record.type : "private",
                rules: [{ required: true }]
              })(
                <Select style={{ maxWidth: 120 }}>
                  <Select.Option value="private">Private</Select.Option>
                  <Select.Option value="public">Public</Select.Option>
                </Select>
              )}
            </Form.Item>
            <Form.Item
              {...formItemLayout}
              name="optin"
              label="Opt-in"
              extra="Double opt-in sends an e-mail to the subscriber asking for confirmation.
              On Double opt-in lists, campaigns are only sent to confirmed subscribers."
            >
              {getFieldDecorator("optin", {
                initialValue: record.optin ? record.optin : "single",
                rules: [{ required: true }]
              })(
                <Select style={{ maxWidth: 120 }}>
                  <Select.Option value="single">Single</Select.Option>
                  <Select.Option value="double">Double</Select.Option>
                </Select>
              )}
            </Form.Item>
            <Form.Item
              {...formItemLayout}
              label="Tags"
              extra="Hit Enter after typing a word to add multiple tags"
            >
              {getFieldDecorator("tags", { initialValue: record.tags })(
                <Select mode="tags" />
              )}
            </Form.Item>
          </Form>
        </Spin>
      </Modal>
    )
  }
}

const CreateForm = Form.create()(CreateFormDef)

class Lists extends React.PureComponent {
  defaultPerPage = 20
  state = {
    formType: null,
    record: {},
    queryParams: {}
  }

  // Pagination config.
  paginationOptions = {
    hideOnSinglePage: false,
    showSizeChanger: true,
    showQuickJumper: true,
    defaultPageSize: this.defaultPerPage,
    pageSizeOptions: ["20", "50", "70", "100"],
    position: "both",
    showTotal: (total, range) => `${range[0]} to ${range[1]} of ${total}`,
    onChange: (page, perPage) => {
      this.fetchRecords({ page: page, per_page: perPage })
    },
    onShowSizeChange: (page, perPage) => {
      this.fetchRecords({ page: page, per_page: perPage })
    }
  }

  constructor(props) {
    super(props)

    this.columns = [
      {
        title: "Name",
        dataIndex: "name",
        sorter: true,
        width: "40%",
        render: (text, record) => {
          const out = []
          out.push(
            <div className="name" key={`name-${record.id}`}>
              <a role="button" onClick={() => this.handleShowEditForm(record)}>
                {text}
              </a>
            </div>
          )

          if (record.tags.length > 0) {
            for (let i = 0; i < record.tags.length; i++) {
              out.push(<Tag key={`tag-${i}`}>{record.tags[i]}</Tag>)
            }
          }

          return out
        }
      },
      {
        title: "Type",
        dataIndex: "type",
        width: "15%",
        render: (type, record) => {
          let color = type === "private" ? "orange" : "green"
          return (
            <div>
              <p>
                <Tag color={color}>{type}</Tag>
                <Tag>{record.optin}</Tag>
              </p>
              {record.optin === cs.ListOptinDouble && (
                <p className="text-small">
                  <Tooltip title="Send a campaign to unconfirmed subscribers to opt-in">
                    <Link to={`/campaigns/new?list_id=${record.id}`}>
                      <Icon type="rocket" /> Send opt-in campaign
                    </Link>
                  </Tooltip>
                </p>
              )}
            </div>
          )
        }
      },
      {
        title: "Subscribers",
        dataIndex: "subscriber_count",
        width: "10%",
        align: "center",
        render: (text, record) => {
          return (
            <div className="name" key={`name-${record.id}`}>
              <Link to={`/subscribers/lists/${record.id}`}>{text}</Link>
            </div>
          )
        }
      },
      {
        title: "Created",
        dataIndex: "created_at",
        render: (date, _) => {
          return Utils.DateString(date)
        }
      },
      {
        title: "Updated",
        dataIndex: "updated_at",
        render: (date, _) => {
          return Utils.DateString(date)
        }
      },
      {
        title: "",
        dataIndex: "actions",
        width: "10%",
        render: (text, record) => {
          return (
            <div className="actions">
              <Tooltip title="Send a campaign">
                <Link to={`/campaigns/new?list_id=${record.id}`}>
                  <Icon type="rocket" />
                </Link>
              </Tooltip>
              <Tooltip title="Edit list">
                <a
                  role="button"
                  onClick={() => this.handleShowEditForm(record)}
                >
                  <Icon type="edit" />
                </a>
              </Tooltip>
              <Popconfirm
                title="Are you sure?"
                onConfirm={() => this.deleteRecord(record)}
              >
                <Tooltip title="Delete list" placement="bottom">
                  <a role="button">
                    <Icon type="delete" />
                  </a>
                </Tooltip>
              </Popconfirm>
            </div>
          )
        }
      }
    ]
  }

  componentDidMount() {
    this.props.pageTitle("Lists")
    this.fetchRecords()
  }

  fetchRecords = params => {
    let qParams = {
      page: this.state.queryParams.page,
      per_page: this.state.queryParams.per_page
    }
    if (params) {
      qParams = { ...qParams, ...params }
    }

    this.props
      .modelRequest(cs.ModelLists, cs.Routes.GetLists, cs.MethodGet, qParams)
      .then(() => {
        this.setState({
          queryParams: {
            ...this.state.queryParams,
            total: this.props.data[cs.ModelLists].total,
            perPage: this.props.data[cs.ModelLists].per_page,
            page: this.props.data[cs.ModelLists].page,
            query: this.props.data[cs.ModelLists].query
          }
        })
      })
  }

  deleteRecord = record => {
    this.props
      .modelRequest(cs.ModelLists, cs.Routes.DeleteList, cs.MethodDelete, {
        id: record.id
      })
      .then(() => {
        notification["success"]({
          placement: cs.MsgPosition,
          message: "List deleted",
          description: `"${record.name}" deleted`
        })

        // Reload the table.
        this.fetchRecords()
      })
      .catch(e => {
        notification["error"]({
          placement: cs.MsgPosition,
          message: "Error",
          description: e.message
        })
      })
  }

  handleHideForm = () => {
    this.setState({ formType: null })
  }

  handleShowCreateForm = () => {
    this.setState({ formType: cs.FormCreate, record: {} })
  }

  handleShowEditForm = record => {
    this.setState({ formType: cs.FormEdit, record: record })
  }

  render() {
    return (
      <section className="content">
        <Row>
          <Col xs={12} sm={18}>
            <h1>Lists ({this.props.data[cs.ModelLists].total}) </h1>
          </Col>
          <Col xs={12} sm={6} className="right">
            <Button
              type="primary"
              icon="plus"
              onClick={this.handleShowCreateForm}
            >
              Create list
            </Button>
          </Col>
        </Row>
        <br />

        <Table
          className="lists"
          columns={this.columns}
          rowKey={record => record.uuid}
          dataSource={(() => {
            if (
              !this.props.data[cs.ModelLists] ||
              !this.props.data[cs.ModelLists].hasOwnProperty("results")
            ) {
              return []
            }
            return this.props.data[cs.ModelLists].results
          })()}
          loading={this.props.reqStates[cs.ModelLists] !== cs.StateDone}
          pagination={{ ...this.paginationOptions, ...this.state.queryParams }}
        />

        <CreateForm
          {...this.props}
          formType={this.state.formType}
          record={this.state.record}
          onClose={this.handleHideForm}
          fetchRecords={this.fetchRecords}
        />
      </section>
    )
  }
}

export default Lists
