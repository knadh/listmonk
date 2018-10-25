import React from "react"
import { Row, Col, Modal, Form, Input, Select, Button, Table, Icon, Tooltip, Tag, Popconfirm, Spin, notification } from "antd"

import Utils from "./utils"
import * as cs from "./constants"


const tagColors = {
    "enabled": "green",
    "blacklisted": "red"
}

class CreateFormDef extends React.PureComponent {
    state = {
        confirmDirty: false,
        attribs: {},
        modalWaiting: false
    }

    componentDidMount() {
        this.setState({ attribs: this.props.record.attribs })
    }

    // Handle create / edit form submission.
    handleSubmit = (e) => {
        e.preventDefault()

        var err = null, values = {}
        this.props.form.validateFields((e, v) => {
            err = e
            values = v
        })
        if(err) {
            return
        }

        values["attribs"] = {}

        let a = this.props.form.getFieldValue("attribs-json")
        if(a && a.length > 0) {
            try {
                values["attribs"] = JSON.parse(a)
                if(values["attribs"] instanceof Array) {
                    notification["error"]({ message: "Invalid JSON type",
                                            description: "Attributes should be a map {} and not an array []" })
                    return
                }
            } catch(e) {
                notification["error"]({ message: "Invalid JSON in attributes", description: e.toString() })
                return
            }
        }

        this.setState({ modalWaiting: true })
        if (this.props.formType === cs.FormCreate) {
            // Add a subscriber.
            this.props.modelRequest(cs.ModelSubscribers, cs.Routes.CreateSubscriber, cs.MethodPost, values).then(() => {
                notification["success"]({ message: "Subscriber added", description: `${values["email"]} added` })
                this.props.fetchRecords()
                this.props.onClose()
                this.setState({ modalWaiting: false })
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
                this.setState({ modalWaiting: false })
            })
        } else {
            // Edit a subscriber.
            delete(values["keys"])
            delete(values["vals"])
            this.props.modelRequest(cs.ModelSubscribers, cs.Routes.UpdateSubscriber, cs.MethodPut, { ...values, id: this.props.record.id }).then(() => {
                notification["success"]({ message: "Subscriber modified", description: `${values["email"]} modified` })
                
                // Reload the table.
                this.props.fetchRecords()
                this.props.onClose()
                this.setState({ modalWaiting: false })
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
                this.setState({ modalWaiting: false })
            })
        }
    }

    modalTitle(formType, record) {
        if(formType === cs.FormCreate) {
            return "Add subscriber"
        }

        return (
             <span>
                 <Tag color={ tagColors.hasOwnProperty(record.status) ? tagColors[record.status] : "" }>{ record.status }</Tag>
                 {" "}
                { record.name } ({ record.email })
             </span>
        )
    }

    render() {
        const { formType, record, onClose } = this.props;
        const { getFieldDecorator } = this.props.form
        const formItemLayout = {
            labelCol: { xs: { span: 16 }, sm: { span: 4 } },
            wrapperCol: { xs: { span: 16 }, sm: { span: 18 } }
        }

        if (formType === null) {
            return null
        }

        let subListIDs = []
        let subStatuses = {}
        if(this.props.record && this.props.record.lists) {
            subListIDs = this.props.record.lists.map((v) => { return v["id"] })
            subStatuses = this.props.record.lists.reduce((o, item) => ({ ...o, [item.id]: item.subscription_status}), {})
        } else if(this.props.list) {
            subListIDs = [ this.props.list.id ]
        }

        return (
            <Modal visible={ true } width="750px"
                className="subscriber-modal"
                title={ this.modalTitle(formType, record) }
                okText={ this.state.form === cs.FormCreate ? "Add" : "Save" }
                confirmLoading={ this.state.modalWaiting }
                onCancel={ onClose }
                onOk={ this.handleSubmit }
                okButtonProps={{ disabled: this.props.reqStates[cs.ModelSubscribers] === cs.StatePending }}>

                <div id="modal-alert-container"></div>
                <Spin spinning={ this.props.reqStates[cs.ModelSubscribers] === cs.StatePending }>
                    <Form onSubmit={this.handleSubmit}>
                        <Form.Item {...formItemLayout} label="E-mail">
                            {getFieldDecorator("email", {
                                initialValue: record.email,
                                rules: [{ required: true }]
                            })(<Input autoFocus pattern="(.+?)@(.+?)" maxLength="200" />)}
                        </Form.Item>
                        <Form.Item {...formItemLayout} label="Name">
                            {getFieldDecorator("name", {
                                initialValue: record.name,
                                rules: [{ required: true }]
                            })(<Input maxLength="200" />)}
                        </Form.Item>
                        <Form.Item {...formItemLayout} name="status" label="Status" extra="Blacklisted users will not receive any e-mails ever">
                            {getFieldDecorator("status", { initialValue: record.status ? record.status : "enabled", rules: [{ required: true, message: "Type is required" }] })(
                                <Select style={{ maxWidth: 120 }}>
                                    <Select.Option value="enabled">Enabled</Select.Option>
                                    <Select.Option value="blacklisted">Blacklisted</Select.Option>
                                </Select>
                            )}
                        </Form.Item>
                        <Form.Item {...formItemLayout} label="Lists" extra="Lists to subscribe to. Lists from which subscribers have unsubscribed themselves cannot be removed.">
                            {getFieldDecorator("lists", { initialValue: subListIDs })(
                                <Select mode="multiple">
                                    {[...this.props.lists].map((v, i) =>
                                        <Select.Option value={ v.id } key={ v.id } disabled={ subStatuses[v.id] === cs.SubscriptionStatusUnsubscribed }>
                                            <span>{ v.name }
                                                { subStatuses[v.id] &&
                                                    <sup className={ "status " + subStatuses[v.id] }> { subStatuses[v.id] }</sup>
                                                }
                                            </span>
                                        </Select.Option>
                                    )}
                                </Select>
                            )}
                        </Form.Item>
                        <section>
                            <h3>Attributes</h3>
                            <p className="ant-form-extra">Attributes can be defined as a JSON map, for example:
                                {'{"age": 30, "color": "red", "is_user": true}'}. <a href="">More info</a>.</p>

                            <div className="json-editor">
                                {getFieldDecorator("attribs-json", {
                                    initialValue: JSON.stringify(this.state.attribs, null, 4)
                                })(
                                <Input.TextArea placeholder="{}"
                                rows={10}
                                readOnly={false}
                                autosize={{ minRows: 5, maxRows: 10 }} />)}
                            </div>
                        </section>
                    </Form>
                </Spin>
            </Modal>
        )
    }
}

const CreateForm = Form.create()(CreateFormDef)

class Subscribers extends React.PureComponent {
    defaultPerPage = 20

    state = {
        formType: null,
        record: {},
        queryParams: {
            page: 1,
            total: 0,
            perPage: this.defaultPerPage,
            listID: this.props.route.match.params.listID ? parseInt(this.props.route.match.params.listID, 10) : 0,
            list: null,
            query: null,
            targetLists: []
        },
        listAddVisible: false
    }

    // Pagination config.
    paginationOptions = {
        hideOnSinglePage: true,
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

        // Table layout.
        this.columns = [{
            title: "E-mail",
            dataIndex: "email",
            sorter: true,
            width: "25%",
            render: (text, record) => {
                return (
                    <a role="button" onClick={() => this.handleShowEditForm(record)}>{text}</a>
                )
            }
        },
        {
            title: "Name",
            dataIndex: "name",
            sorter: true,
            width: "25%",
            render: (text, record) => {
                return (
                    <a role="button" onClick={() => this.handleShowEditForm(record)}>{text}</a>
                )
            }
        },
        {
            title: "Status",
            dataIndex: "status",
            width: "5%",
            render: (status, _) => {
                return <Tag color={ tagColors.hasOwnProperty(status) ? tagColors[status] : "" }>{ status }</Tag>
            }
        },
        {
            title: "Lists",
            dataIndex: "lists",
            width: "10%",
            align: "center",
            render: (lists, _) => {
                return <span>{ lists.reduce((def, item) => def + (item.subscription_status !== cs.SubscriptionStatusUnsubscribed ? 1 : 0), 0) }</span>
            }
        },
        {
            title: "Created",
            width: "10%",
            dataIndex: "created_at",
            render: (date, _) => {
                return Utils.DateString(date)
            }
        },
        {
            title: "Updated",
            width: "10%",
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
                        {/* <Tooltip title="Send an e-mail"><a role="button"><Icon type="rocket" /></a></Tooltip> */}
                        <Tooltip title="Edit subscriber"><a role="button" onClick={() => this.handleShowEditForm(record)}><Icon type="edit" /></a></Tooltip>
                        <Popconfirm title="Are you sure?" onConfirm={() => this.handleDeleteRecord(record)}>
                            <Tooltip title="Delete subscriber" placement="bottom"><a role="button"><Icon type="delete" /></a></Tooltip>
                        </Popconfirm>
                    </div>
                )
            }
        }
        ]
    }

    componentDidMount() {
        // Load lists on boot.
        this.props.modelRequest(cs.ModelLists, cs.Routes.GetLists, cs.MethodGet).then(() => {
            // If this is an individual list's view, pick up that list.
            if(this.state.queryParams.listID) {
                this.props.data[cs.ModelLists].forEach((l) => {
                    if(l.id === this.state.queryParams.listID) {
                        this.setState({ queryParams: { ...this.state.queryParams, list: l }})
                        return false
                    }
                })
            }
        })

        this.fetchRecords()
    }

    fetchRecords = (params) => {
        let qParams = {
            page: this.state.queryParams.page,
            per_page: this.state.queryParams.per_page,
            list_id: this.state.queryParams.listID,
            query: this.state.queryParams.query
        }

        // The records are for a specific list.
        if(this.state.queryParams.listID) {
            qParams.list_id = this.state.queryParams.listID
        }

        if(params) {
            qParams = { ...qParams, ...params }
        }

        this.props.modelRequest(cs.ModelSubscribers, cs.Routes.GetSubscribers, cs.MethodGet, qParams).then(() => {
            this.setState({ queryParams: {
                ...this.state.queryParams,
                total: this.props.data[cs.ModelSubscribers].total,
                perPage: this.props.data[cs.ModelSubscribers].per_page,
                page: this.props.data[cs.ModelSubscribers].page,
                query: this.props.data[cs.ModelSubscribers].query,
            }})
        })
    }

    handleDeleteRecord = (record) => {
        this.props.modelRequest(cs.ModelSubscribers, cs.Routes.DeleteSubscriber, cs.MethodDelete, { id: record.id })
            .then(() => {
                notification["success"]({ message: "Subscriber deleted", description: `${record.email} deleted` })

                // Reload the table.
                this.fetchRecords()
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
            })
    }

    handleQuerySubscribersIntoLists = (query, sourceList, targetLists) => {
        let params = {
            query: query,
            source_list: sourceList,
            target_lists: targetLists
        }

        this.props.request(cs.Routes.QuerySubscribersIntoLists, cs.MethodPost, params).then((res) => {
            notification["success"]({ message: "Subscriber(s) added", description: `${ res.data.data.count } added` })
            this.handleToggleListAdd()
        }).catch(e => {
            notification["error"]({ message: "Error", description: e.message })
        })
    }

    handleHideForm = () => {
        this.setState({ formType: null })
    }

    handleShowCreateForm = () => {
        this.setState({ formType: cs.FormCreate, attribs: [], record: {} })
    }

    handleShowEditForm = (record) => {
        this.setState({ formType: cs.FormEdit, record: record })
    }

    handleToggleQueryForm = () => {
        // The query form is being cancelled. Reset the results.
        if(this.state.queryFormVisible) {
            this.fetchRecords({
                query: null
            })
        }

        this.setState({ queryFormVisible: !this.state.queryFormVisible })
    }

    handleToggleListAdd = () => {
        this.setState({ listAddVisible: !this.state.listAddVisible })
    }

    render() {
        const pagination = {
            ...this.paginationOptions,
            ...this.state.queryParams
        }

        return (
            <section className="content">
                <header className="header">
                    <Row>
                    <Col span={ 20 }>
                        <h1>
                            Subscribers
                            { this.state.queryParams.list &&
                                <span> &raquo; { this.state.queryParams.list.name }</span> }

                        </h1>
                    </Col>
                    <Col span={ 2 }>
                        { !this.state.queryFormVisible &&
                        <a role="button" onClick={ this.handleToggleQueryForm }><Icon type="search" /> Advanced</a> }
                     </Col>
                    <Col span={ 2 }>
                        <Button type="primary" icon="plus" onClick={ this.handleShowCreateForm }>Add subscriber</Button>
                    </Col>
                    </Row>
                </header>
                
                { this.state.queryFormVisible &&
                    <div className="subscriber-query">
                        <p>
                            Write a partial SQL expression to query the subscribers based on their
                            primary information or attributes. Learn more.
                        </p>
                        <Input.TextArea placeholder="name LIKE '%user%'"
                            id="subscriber-query"
                            rows={ 10 }
                            onChange={(e) => {
                                this.setState({ queryParams: { ...this.state.queryParams, query: e.target.value } })
                            }}
                            autosize={{ minRows: 2, maxRows: 10 }} />

                        <div className="actions">
                            <Button
                                disabled={ this.state.queryParams.query === "" }
                                type="primary"
                                icon="search"
                                onClick={ () => { this.fetchRecords() } }>Query</Button>
                            {" "}
                            <Button
                                disabled={ !this.state.queryParams.total }
                                icon="plus"
                                onClick={ this.handleToggleListAdd }>Add ({this.state.queryParams.total}) to list</Button>
                            {" "}
                            <Button icon="close" onClick={ this.handleToggleQueryForm }>Cancel</Button>
                        </div>
                    </div>
                }

                <Table
                    columns={ this.columns }
                    rowKey={ record => `${record.id}-${record.email}` }
                    dataSource={ this.props.data[cs.ModelSubscribers].results }
                    loading={ this.props.reqStates[cs.ModelSubscribers] !== cs.StateDone }
                    pagination={ pagination }
                    rowSelection = {{
                        fixed: true
                    }}
                />

                { this.state.formType !== null && <CreateForm {...this.props}
                    formType={ this.state.formType }
                    record={ this.state.record }
                    lists={ this.props.data[cs.ModelLists] }
                    list={ this.state.queryParams.list }
                    fetchRecords={ this.fetchRecords }
                    queryParams= { this.state.queryParams }
                    onClose={ this.handleHideForm } />
                }

                <Modal visible={ this.state.listAddVisible } width="750px"
                    className="list-add-modal"
                    title={ "Add " + this.props.data[cs.ModelSubscribers].total + " subscriber(s) to lists" }
                    okText="Add"
                    onCancel={ this.handleToggleListAdd }
                    onOk={() => {
                        if(this.state.queryParams.targetLists.length == 0) {
                            notification["warning"]({
                                message: "No lists selected",
                                description: "Select one or more lists"
                            })
                            return false
                        }

                        this.handleQuerySubscribersIntoLists(
                            this.state.queryParams.query,
                            this.state.queryParams.listID,
                            this.state.queryParams.targetLists
                        )
                    }}
                    okButtonProps={{ disabled: this.props.reqStates[cs.ModelSubscribers] === cs.StatePending }}>
                        <Select mode="multiple" style={{ width: "100%" }} onChange={(lists) => {
                            this.setState({ queryParams: { ...this.state.queryParams, targetLists: lists} })
                        }}>
                            { this.props.data[cs.ModelLists].map((v, i) =>
                                <Select.Option value={ v.id } key={ v.id }>{ v.name }</Select.Option>
                            )}
                        </Select>
                </Modal>
            </section>
        )
    }
}

export default Subscribers
