import React from "react"
import { Link } from "react-router-dom"
import { Row, Col, Modal, Form, Input, Select, Button, Table, Icon, Tooltip, Tag, Popconfirm, Spin, notification, Radio } from "antd"

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
                    <Form onSubmit={ this.handleSubmit }>
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


class ListsFormDef extends React.PureComponent {
    state = {
        modalWaiting: false
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

        if(this.props.allRowsSelected) {
            values["list_ids"] = this.props.listIDs
            values["query"] = this.props.query
        } else {
            values["ids"] = this.props.selectedRows.map(r => r.id)
        }

        this.setState({ modalWaiting: true })
        this.props.request(!this.props.allRowsSelected ? cs.Routes.AddSubscribersToLists : cs.Routes.AddSubscribersToListsByQuery,
                           cs.MethodPut, values).then(() => {
            notification["success"]({ message: "Lists changed",
                                      description: `Lists changed for selected subscribers` })
            this.props.clearSelectedRows()
            this.props.fetchRecords()
            this.setState({ modalWaiting: false })
            this.props.onClose()
        }).catch(e => {
            notification["error"]({ message: "Error", description: e.message })
            this.setState({ modalWaiting: false })
        })
    }

    render() {
        const { getFieldDecorator } = this.props.form
        const formItemLayout = {
            labelCol: { xs: { span: 16 }, sm: { span: 4 } },
            wrapperCol: { xs: { span: 16 }, sm: { span: 18 } }
        }

        return (
            <Modal visible={ true } width="750px"
                className="subscriber-lists-modal"
                title="Manage lists"
                okText="Ok"
                confirmLoading={ this.state.modalWaiting }
                onCancel={ this.props.onClose }
                onOk={ this.handleSubmit }>
                <Form onSubmit={ this.handleSubmit }>
                    <Form.Item {...formItemLayout} label="Action">
                        {getFieldDecorator("action", {
                            initialValue: "add",
                            rules: [{ required: true }]
                        })(
                            <Radio.Group>
                                <Radio value="add">Add</Radio>
                                <Radio value="remove">Remove</Radio>
                                <Radio value="unsubscribe">Mark as unsubscribed</Radio>
                            </Radio.Group>
                        )}
                    </Form.Item>
                    <Form.Item {...formItemLayout} label="Lists">
                        {getFieldDecorator("target_list_ids", { rules:[{ required: true }] })(
                            <Select mode="multiple">
                                {[...this.props.lists].map((v, i) =>
                                    <Select.Option value={ v.id } key={ v.id }>
                                        { v.name }
                                    </Select.Option>
                                )}
                            </Select>
                        )}
                    </Form.Item>
                </Form>
            </Modal>
        )
    }
}

const CreateForm = Form.create()(CreateFormDef)
const ListsForm = Form.create()(ListsFormDef)

class Subscribers extends React.PureComponent {
    defaultPerPage = 20

    state = {
        formType: null,
        listsFormVisible: false,
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
        listModalVisible: false,
        allRowsSelected: false,
        selectedRows: []
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
                const out = [];
                out.push(
                    <div key={`sub-email-${ record.id }`} className="sub-name">
                        <a role="button" onClick={() => { this.handleShowEditForm(record)}}>{text}</a>
                    </div>
                )
                
                if(record.lists.length > 0) {
                    for (let i = 0; i < record.lists.length; i++) {
                        out.push(<Tag className="list" key={`sub-${ record.id }-list-${ record.lists[i].id }`}>
                            <Link to={ `/subscribers/lists/${ record.lists[i].id }` }>{ record.lists[i].name }</Link>
                        </Tag>)
                    }
                }

                return out
            }
        },
        {
            title: "Name",
            dataIndex: "name",
            sorter: true,
            width: "15%",
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

    handleDeleteRecords = (records) => {
        this.props.modelRequest(cs.ModelSubscribers, cs.Routes.DeleteSubscribers, cs.MethodDelete, { id: records.map(r => r.id) })
            .then(() => {
                notification["success"]({ message: "Subscriber(s) deleted", description: "Selected subscribers deleted" })

                // Reload the table.
                this.fetchRecords()
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
            })
    }

    handleBlacklistSubscribers = (records) => {
        this.props.request(cs.Routes.BlacklistSubscribers, cs.MethodPut, { ids: records.map(r => r.id) })
            .then(() => {
                notification["success"]({ message: "Subscriber(s) blacklisted", description: "Selected subscribers blacklisted" })

                // Reload the table.
                this.fetchRecords()
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
            })
    }

    // Arbitrary query based calls.
    handleDeleteRecordsByQuery = (listIDs, query) => {
        this.props.modelRequest(cs.ModelSubscribers, cs.Routes.DeleteSubscribersByQuery, cs.MethodPost,
            { list_ids: listIDs, query: query })
            .then(() => {
                notification["success"]({ message: "Subscriber(s) deleted", description: "Selected subscribers have been deleted" })

                // Reload the table.
                this.fetchRecords()
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
            })
    }

    handleBlacklistSubscribersByQuery = (listIDs, query) => {
        this.props.request(cs.Routes.BlacklistSubscribersByQuery, cs.MethodPut,
            { list_ids: listIDs, query: query })
            .then(() => {
                notification["success"]({ message: "Subscriber(s) blacklisted", description: "Selected subscribers have been blacklisted" })

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
            this.handleToggleListModal()
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

    handleToggleListsForm = () => {
        this.setState({ listsFormVisible: !this.state.listsFormVisible })
    }

    handleSearch = (q) => {
        q = q.trim().toLowerCase()
        if(q === "") {
            this.fetchRecords({ query: null })
            return
        }

        q = q.replace(/'/g, "''")
        const query = `(name LIKE '%${q}%' OR email LIKE '%${q}%')`
        this.fetchRecords({ query: query })
    }

    handleSelectRow = (_, records) => {
        this.setState({ allRowsSelected: false, selectedRows: records })
    }

    handleSelectAllRows = () => {
        this.setState({ allRowsSelected: true,
                        selectedRows: this.props.data[cs.ModelSubscribers].results })
    }

    clearSelectedRows = (_, records) => {
        this.setState({ allRowsSelected: false, selectedRows: [] })
    }

    handleToggleQueryForm = () => {
        this.setState({ queryFormVisible: !this.state.queryFormVisible })
    }

    handleToggleListModal = () => {
        this.setState({ listModalVisible: !this.state.listModalVisible })
    }

    render() {
        const pagination = {
            ...this.paginationOptions,
            ...this.state.queryParams
        }

        if(this.state.queryParams.list) {
            this.props.pageTitle(this.state.queryParams.list.name + " / Subscribers")
        } else {
            this.props.pageTitle("Subscribers")
        }

        return (
            <section className="content">
                <header className="header">
                    <Row>
                        <Col span={ 20 }>
                            <h1>
                                Subscribers
                                { this.props.data[cs.ModelSubscribers].total > 0 &&
                                    <span> ({ this.props.data[cs.ModelSubscribers].total })</span> }
                                { this.state.queryParams.list &&
                                    <span> &raquo; { this.state.queryParams.list.name }</span> }
                            </h1>
                        </Col>
                        <Col span={ 2 }>
                            <Button type="primary" icon="plus" onClick={ this.handleShowCreateForm }>Add subscriber</Button>
                        </Col>
                    </Row>
                </header>
                
                <div className="subscriber-query">
                    <Row>
                        <Col span={10}>
                            <Row>
                                <Col span={ 15 }>
                                    <label>Search subscribers</label>
                                    <Input.Search
                                        name="name"
                                        placeholder="Name or e-mail"
                                        enterButton
                                        onSearch={ this.handleSearch } />
                                    {" "}
                                </Col>
                                <Col span={ 8 } offset={ 1 }>
                                    <label>&nbsp;</label><br />
                                    <a role="button" onClick={ this.handleToggleQueryForm }>
                                        <Icon type="setting" /> Advanced</a>
                                </Col>
                            </Row>
                            { this.state.queryFormVisible &&
                                <div className="advanced-query">
                                    <p>
                                        <label>Advanced query</label>
                                        <Input.TextArea placeholder="name LIKE '%user%'"
                                            id="subscriber-query"
                                            rows={ 10 }
                                            onChange={(e) => {
                                                this.setState({ queryParams: { ...this.state.queryParams, query: e.target.value } })
                                            }}
                                            value={ this.state.queryParams.query }
                                            autosize={{ minRows: 2, maxRows: 10 }} />
                                        <span className="text-tiny text-small">
                                            Write a partial SQL expression to query the subscribers based on their primary information or attributes. Learn more.
                                        </span>
                                    </p>
                                    <p>
                                        <Button
                                            disabled={ this.state.queryParams.query === "" }
                                            type="primary"
                                            icon="search"
                                            onClick={ () => { this.fetchRecords() } }>Query</Button>
                                        {" "}
                                        <Button
                                            disabled={ this.state.queryParams.query === "" }
                                            icon="refresh"
                                            onClick={ () => { this.fetchRecords({ query: null }) } }>Reset</Button>
                                    </p>
                                </div>
                            }
                        </Col>
                        <Col span={14}>
                            { this.state.selectedRows.length > 0 &&
                                <nav className="table-options">
                                    <p>
                                        <strong>{ this.state.allRowsSelected ? this.state.queryParams.total : this.state.selectedRows.length }</strong>
                                        {" "} subscriber(s) selected
                                        { !this.state.allRowsSelected && this.state.queryParams.total > this.state.queryParams.perPage &&
                                            <span> &mdash; <a role="button" onClick={ this.handleSelectAllRows }>
                                                                Select all { this.state.queryParams.total }?</a>
                                            </span>
                                        }
                                    </p>
                                    <p>
                                        <a role="button" onClick={ this.handleToggleListsForm }>
                                            <Icon type="bars" /> Manage lists
                                        </a>
                                        <a role="button"><Icon type="rocket" /> Send campaign</a>
                                        <Popconfirm title="Are you sure?" onConfirm={() => {
                                                if(this.state.allRowsSelected) {
                                                    this.handleDeleteRecordsByQuery(this.state.queryParams.listID ? [this.state.queryParams.listID] : [], this.state.queryParams.query)
                                                    this.clearSelectedRows()
                                                } else {
                                                    this.handleDeleteRecords(this.state.selectedRows)
                                                    this.clearSelectedRows()
                                                }
                                            }}>
                                            <a role="button"><Icon type="delete" /> Delete</a>
                                        </Popconfirm>
                                        <Popconfirm title="Are you sure?" onConfirm={() => {
                                                if(this.state.allRowsSelected) {
                                                    this.handleBlacklistSubscribersByQuery(this.state.queryParams.listID ? [this.state.queryParams.listID] : [], this.state.queryParams.query)
                                                    this.clearSelectedRows()
                                                } else {
                                                    this.handleBlacklistSubscribers(this.state.selectedRows)
                                                    this.clearSelectedRows()
                                                }
                                            }}>
                                            <a role="button"><Icon type="close" /> Blacklist</a>
                                        </Popconfirm>
                                    </p>
                                </nav>
                            }
                        </Col>
                    </Row>
                </div>

                <Table
                    columns={ this.columns }
                    rowKey={ record => `sub-${record.id}` }
                    dataSource={ this.props.data[cs.ModelSubscribers].results }
                    loading={ this.props.reqStates[cs.ModelSubscribers] !== cs.StateDone }
                    pagination={ pagination }
                    rowSelection = {{
                        columnWidth: "5%",
                        onChange: this.handleSelectRow,
                        selectedRowKeys: this.state.selectedRows.map(r => `sub-${r.id}`)
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

                { this.state.listsFormVisible && <ListsForm {...this.props}
                    lists={ this.props.data[cs.ModelLists] }
                    allRowsSelected={ this.state.allRowsSelected }
                    selectedRows={ this.state.selectedRows }
                    selectedLists={ this.state.queryParams.listID ? [this.state.queryParams.listID] : []}
                    clearSelectedRows={ this.clearSelectedRows }
                    query={ this.state.queryParams.query }
                    fetchRecords={ this.fetchRecords }
                    onClose={ this.handleToggleListsForm } />
                }
            </section>
        )
    }
}

export default Subscribers
