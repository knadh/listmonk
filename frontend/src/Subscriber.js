import React from "react";
import {
  Row,
  Col,
  Form,
  Input,
  Select,
  Button,
  Tag,
  Spin,
  Popconfirm,
  notification
} from "antd";

import * as cs from "./constants";

const tagColors = {
  enabled: "green",
  blacklisted: "red"
};
const formItemLayoutModal = {
  labelCol: { xs: { span: 24 }, sm: { span: 4 } },
  wrapperCol: { xs: { span: 24 }, sm: { span: 18 } }
};
const formItemLayout = {
  labelCol: { xs: { span: 16 }, sm: { span: 4 } },
  wrapperCol: { xs: { span: 16 }, sm: { span: 10 } }
};
const formItemTailLayout = {
  wrapperCol: { xs: { span: 24, offset: 0 }, sm: { span: 10, offset: 4 } }
};

class CreateFormDef extends React.PureComponent {
  state = {
    confirmDirty: false,
    loading: false
  };

  // Handle create / edit form submission.
  handleSubmit = (e, cb) => {
    e.preventDefault();
    if (!cb) {
      // Set a fake callback.
      cb = () => {};
    }

    var err = null,
      values = {};
    this.props.form.validateFields((e, v) => {
      err = e;
      values = v;
    });
    if (err) {
      return;
    }

    let a = values["attribs"];
    values["attribs"] = {};
    if (a && a.length > 0) {
      try {
        values["attribs"] = JSON.parse(a);
        if (values["attribs"] instanceof Array) {
          notification["error"]({
            message: "Invalid JSON type",
            description: "Attributes should be a map {} and not an array []"
          });
          return;
        }
      } catch (e) {
        notification["error"]({
          message: "Invalid JSON in attributes",
          description: e.toString()
        });
        return;
      }
    }

    this.setState({ loading: true });
    if (this.props.formType === cs.FormCreate) {
      // Add a subscriber.
      this.props
        .modelRequest(
          cs.ModelSubscribers,
          cs.Routes.CreateSubscriber,
          cs.MethodPost,
          values
        )
        .then(() => {
          notification["success"]({
            message: "Subscriber added",
            description: `${values["email"]} added`
          });
          if (!this.props.isModal) {
            this.props.fetchRecord(this.props.record.id);
          }
          cb(true);
          this.setState({ loading: false });
        })
        .catch(e => {
          notification["error"]({ message: "Error", description: e.message });
          cb(false);
          this.setState({ loading: false });
        });
    } else {
      // Edit a subscriber.
      delete values["keys"];
      delete values["vals"];
      this.props
        .modelRequest(
          cs.ModelSubscribers,
          cs.Routes.UpdateSubscriber,
          cs.MethodPut,
          { ...values, id: this.props.record.id }
        )
        .then(resp => {
          notification["success"]({
            message: "Subscriber modified",
            description: `${values["email"]} modified`
          });
          if (!this.props.isModal) {
            this.props.fetchRecord(this.props.record.id);
          }
          cb(true);
          this.setState({ loading: false });
        })
        .catch(e => {
          notification["error"]({ message: "Error", description: e.message });
          cb(false);
          this.setState({ loading: false });
        });
    }
  };

  handleDeleteRecord = record => {
    this.props
      .modelRequest(
        cs.ModelSubscribers,
        cs.Routes.DeleteSubscriber,
        cs.MethodDelete,
        { id: record.id }
      )
      .then(() => {
        notification["success"]({
          message: "Subscriber deleted",
          description: `${record.email} deleted`
        });

        this.props.route.history.push({
          pathname: cs.Routes.ViewSubscribers
        });
      })
      .catch(e => {
        notification["error"]({ message: "Error", description: e.message });
      });
  };

  render() {
    const { formType, record } = this.props;
    const { getFieldDecorator } = this.props.form;

    if (formType === null) {
      return null;
    }

    let subListIDs = [];
    let subStatuses = {};
    if (this.props.record && this.props.record.lists) {
      subListIDs = this.props.record.lists.map(v => {
        return v["id"];
      });
      subStatuses = this.props.record.lists.reduce(
        (o, item) => ({ ...o, [item.id]: item.subscription_status }),
        {}
      );
    } else if (this.props.list) {
      subListIDs = [this.props.list.id];
    }

    const layout = this.props.isModal ? formItemLayoutModal : formItemLayout;
    return (
      <Spin spinning={this.state.loading}>
        <Form onSubmit={this.handleSubmit}>
          <Form.Item {...layout} label="E-mail">
            {getFieldDecorator("email", {
              initialValue: record.email,
              rules: [{ required: true }]
            })(<Input autoFocus pattern="(.+?)@(.+?)" maxLength={200} />)}
          </Form.Item>
          <Form.Item {...layout} label="Name">
            {getFieldDecorator("name", {
              initialValue: record.name,
              rules: [{ required: true }]
            })(<Input maxLength={200} />)}
          </Form.Item>
          <Form.Item
            {...layout}
            name="status"
            label="Status"
            extra="Blacklisted users will not receive any e-mails ever"
          >
            {getFieldDecorator("status", {
              initialValue: record.status ? record.status : "enabled",
              rules: [{ required: true, message: "Type is required" }]
            })(
              <Select style={{ maxWidth: 120 }}>
                <Select.Option value="enabled">Enabled</Select.Option>
                <Select.Option value="blacklisted">Blacklisted</Select.Option>
              </Select>
            )}
          </Form.Item>
          <Form.Item
            {...layout}
            label="Lists"
            extra="Lists to subscribe to. Lists from which subscribers have unsubscribed themselves cannot be removed."
          >
            {getFieldDecorator("lists", { initialValue: subListIDs })(
              <Select mode="multiple">
                {[...this.props.lists].map((v, i) => (
                  <Select.Option
                    value={v.id}
                    key={v.id}
                    disabled={
                      subStatuses[v.id] === cs.SubscriptionStatusUnsubscribed
                    }
                  >
                    <span>
                      {v.name}
                      {subStatuses[v.id] && (
                        <sup
                          className={"subscription-status " + subStatuses[v.id]}
                        >
                          {" "}
                          {subStatuses[v.id]}
                        </sup>
                      )}
                    </span>
                  </Select.Option>
                ))}
              </Select>
            )}
          </Form.Item>
          <Form.Item {...layout} label="Attributes" colon={false}>
            <div>
              {getFieldDecorator("attribs", {
                initialValue: record.attribs
                  ? JSON.stringify(record.attribs, null, 4)
                  : ""
              })(
                <Input.TextArea
                  placeholder="{}"
                  rows={10}
                  readOnly={false}
                  autosize={{ minRows: 5, maxRows: 10 }}
                />
              )}
            </div>
            <p className="ant-form-extra">
              Attributes are defined as a JSON map, for example:
              {' {"age": 30, "color": "red", "is_user": true}'}.{" "}
              <a href="">More info</a>.
            </p>
          </Form.Item>
          {!this.props.isModal && (
            <Form.Item {...formItemTailLayout}>
              <Button
                type="primary"
                htmlType="submit"
                icon={this.props.formType === cs.FormCreate ? "plus" : "save"}
              >
                {this.props.formType === cs.FormCreate ? "Add" : "Save"}
              </Button>{" "}
              {this.props.formType === cs.FormEdit && (
                <Popconfirm
                  title="Are you sure?"
                  onConfirm={() => {
                    this.handleDeleteRecord(record);
                  }}
                >
                  <Button icon="delete">Delete</Button>
                </Popconfirm>
              )}
            </Form.Item>
          )}
        </Form>
      </Spin>
    );
  }
}

const CreateForm = Form.create()(CreateFormDef);

class Subscriber extends React.PureComponent {
  state = {
    loading: true,
    formRef: null,
    record: {},
    subID: this.props.route.match.params
      ? parseInt(this.props.route.match.params.subID, 10)
      : 0
  };

  componentDidMount() {
    // When this component is invoked within a modal from the subscribers list page,
    // the necessary context is supplied and there's no need to fetch anything.
    if (!this.props.isModal) {
      // Fetch lists.
      this.props.modelRequest(cs.ModelLists, cs.Routes.GetLists, cs.MethodGet);

      // Fetch subscriber.
      this.fetchRecord(this.state.subID);
    } else {
      this.setState({ record: this.props.record, loading: false });
    }
  }

  fetchRecord = id => {
    this.props
      .request(cs.Routes.GetSubscriber, cs.MethodGet, { id: id })
      .then(r => {
        this.setState({ record: r.data.data, loading: false });
      })
      .catch(e => {
        notification["error"]({
          placement: cs.MsgPosition,
          message: "Error",
          description: e.message
        });
      });
  };

  setFormRef = r => {
    this.setState({ formRef: r });
  };

  submitForm = (e, cb) => {
    if (this.state.formRef) {
      this.state.formRef.handleSubmit(e, cb);
    }
  };

  render() {
    return (
      <section className="content">
        <header className="header">
          <Row>
            <Col span={20}>
              {!this.state.record.id && <h1>Add subscriber</h1>}
              {this.state.record.id && (
                <div>
                  <h1>
                    <Tag
                      color={
                        tagColors.hasOwnProperty(this.state.record.status)
                          ? tagColors[this.state.record.status]
                          : ""
                      }
                    >
                      {this.state.record.status}
                    </Tag>{" "}
                    {this.state.record.name} ({this.state.record.email})
                  </h1>
                  <span className="text-small text-grey">
                    ID {this.state.record.id} / UUID {this.state.record.uuid}
                  </span>
                </div>
              )}
            </Col>
            <Col span={2} />
          </Row>
        </header>
        <div>
          <Spin spinning={this.state.loading}>
            <CreateForm
              {...this.props}
              formType={this.props.formType ? this.props.formType : cs.FormEdit}
              record={this.state.record}
              fetchRecord={this.fetchRecord}
              lists={this.props.data[cs.ModelLists].results}
              wrappedComponentRef={r => {
                if (!r) {
                  return;
                }

                // Save the form's reference so that when this component
                // is used as a modal, the invoker of the model can submit
                // it via submitForm()
                this.setState({ formRef: r });
              }}
            />
          </Spin>
        </div>
      </section>
    );
  }
}

export default Subscriber;
