import React from "react"
import { Row, Col, Modal, Form, Input, Button, Table, Icon, Tooltip, Tag, Popconfirm, Spin, notification } from "antd"

import Utils from "./utils"
import * as cs from "./constants"

class CreateFormDef extends React.PureComponent {
    state = {
        confirmDirty: false,
        modalWaiting: false
    }

    // Handle create / edit form submission.
    handleSubmit = (e) => {
        e.preventDefault()
        this.props.form.validateFields((err, values) => {
            if (err) {
                return
            }

            this.setState({ modalWaiting: true })
            if (this.props.formType === cs.FormCreate) {
                // Create a new list.
                this.props.modelRequest(cs.ModelTemplates, cs.Routes.CreateTemplate, cs.MethodPost, values).then(() => {
                    notification["success"]({ placement: "topRight", message: "Template added", description: `"${values["name"]}" added` })
                    this.props.fetchRecords()
                    this.props.onClose()
                    this.setState({ modalWaiting: false })
                }).catch(e => {
                    notification["error"]({ message: "Error", description: e.message })
                    this.setState({ modalWaiting: false })
                })
            } else {
                // Edit a list.
                this.props.modelRequest(cs.ModelTemplates, cs.Routes.UpdateTemplate, cs.MethodPut, { ...values, id: this.props.record.id }).then(() => {
                    notification["success"]({ placement: "topRight", message: "Template updated", description: `"${values["name"]}" modified` })
                    this.props.fetchRecords()
                    this.props.onClose()
                    this.setState({ modalWaiting: false })
                }).catch(e => {
                    notification["error"]({ message: "Error", description: e.message })
                    this.setState({ modalWaiting: false })
                })
            }
        })
    }

    handleConfirmBlur = (e) => {
        const value = e.target.value
        this.setState({ confirmDirty: this.state.confirmDirty || !!value })
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
            <Modal visible={ true } title={ formType === cs.FormCreate ? "Add template" : record.name }
                okText={ this.state.form === cs.FormCreate ? "Add" : "Save" }
                width="90%"
                height={ 900 }
                confirmLoading={ this.state.modalWaiting }
                onCancel={ onClose }
                onOk={ this.handleSubmit }>

                <Spin spinning={ this.props.reqStates[cs.ModelTemplates] === cs.StatePending }>
                    <Form onSubmit={this.handleSubmit}>
                        <Form.Item {...formItemLayout} label="Name">
                            {getFieldDecorator("name", {
                                initialValue: record.name,
                                rules: [{ required: true }]
                            })(<Input autoFocus maxLength="200" />)}
                        </Form.Item>
                        <Form.Item {...formItemLayout} name="body" label="Raw HTML">
                            {getFieldDecorator("body", { initialValue: record.body ? record.body : "", rules: [{ required: true }] })(
                                <Input.TextArea autosize={{ minRows: 10, maxRows: 30 }}>
                                </Input.TextArea>
                            )}
                        </Form.Item>
                    </Form>
                </Spin>
                <Row>
                    <Col span="4"></Col>
                    <Col span="18" className="text-grey text-small">
                        The placeholder <code>{'{'}{'{'} template "content" . {'}'}{'}'}</code> should appear in the template. <a href="" target="_blank">Read more on templating</a>.
                    </Col>
                </Row>
            </Modal>
        )
    }
}

const CreateForm = Form.create()(CreateFormDef)

class Templates extends React.PureComponent {
    state = {
        formType: null,
        record: {},
        previewRecord: null
    }

    constructor(props) {
        super(props)

        this.columns = [{
            title: "Name",
            dataIndex: "name",
            sorter: true,
            width: "50%",
            render: (text, record) => {
                return (
                    <div className="name">
                        <a role="button" onClick={() => this.handleShowEditForm(record)}>{ text }</a>
                        { record.is_default && 
                            <div><Tag>Default</Tag></div>}
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
            width: "20%",
            className: "actions",
            render: (text, record) => {
                return (
                    <div className="actions">
                        <Tooltip title="Preview template" onClick={() => this.handlePreview(record)}><a role="button"><Icon type="search" /></a></Tooltip>

                        { !record.is_default &&
                            <Popconfirm title="Are you sure?" onConfirm={() => this.handleSetDefault(record)}>
                                <Tooltip title="Set as default" placement="bottom"><a role="button"><Icon type="check" /></a></Tooltip>
                            </Popconfirm>
                        }

                        <Tooltip title="Edit template"><a role="button" onClick={() => this.handleShowEditForm(record)}><Icon type="edit" /></a></Tooltip>

                        { record.id !== 1 &&
                            <Popconfirm title="Are you sure?" onConfirm={() => this.handleDeleteRecord(record)}>
                                <Tooltip title="Delete template" placement="bottom"><a role="button"><Icon type="delete" /></a></Tooltip>
                            </Popconfirm>
                        }
                    </div>
                )
            }
        }]
    }

    componentDidMount() {
        this.fetchRecords()
    }

    fetchRecords = () => {
        this.props.modelRequest(cs.ModelTemplates, cs.Routes.GetTemplates, cs.MethodGet)
    }

    handleDeleteRecord = (record) => {
        this.props.modelRequest(cs.ModelTemplates, cs.Routes.DeleteTemplate, cs.MethodDelete, { id: record.id })
            .then(() => {
                notification["success"]({ placement: "topRight", message: "Template deleted", description: `"${record.name}" deleted` })

                // Reload the table.
                this.fetchRecords()
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
            })
    }

    handleSetDefault = (record) => {
        this.props.modelRequest(cs.ModelTemplates, cs.Routes.SetDefaultTemplate, cs.MethodPut, { id: record.id })
            .then(() => {
                notification["success"]({ placement: "topRight", message: "Template updated", description: `"${record.name}" set as default` })
                
                // Reload the table.
                this.fetchRecords()
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
            })
    }

    handlePreview = (record) => {
        this.setState({ previewRecord: record })
    }

    hideForm = () => {
        this.setState({ formType: null })
    }

    handleShowCreateForm = () => {
        this.setState({ formType: cs.FormCreate, record: {} })
    }

    handleShowEditForm = (record) => {
        this.setState({ formType: cs.FormEdit, record: record })
    }

    render() {
        return (
            <section className="content templates">
                <Row>
                    <Col span={22}><h1>Templates ({this.props.data[cs.ModelTemplates].length}) </h1></Col>
                    <Col span={2}>
                        <Button type="primary" icon="plus" onClick={ this.handleShowCreateForm }>Add template</Button>
                    </Col>
                </Row>
                <br />

                <Table
                    columns={ this.columns }
                    rowKey={ record => record.id }
                    dataSource={ this.props.data[cs.ModelTemplates] }
                    loading={ this.props.reqStates[cs.ModelTemplates] !== cs.StateDone }
                    pagination={ false }
                />

                <CreateForm { ...this.props }
                    formType={ this.state.formType }
                    record={ this.state.record }
                    onClose={ this.hideForm }
                    fetchRecords = { this.fetchRecords }
                />

                <Modal visible={ this.state.previewRecord !== null } title={ this.state.previewRecord ? this.state.previewRecord.name : "" }
                    className="template-preview-modal"
                    width="90%"
                    height={ 900 }
                    onOk={ () => { this.setState({ previewRecord: null }) } }>
                    { this.state.previewRecord !== null &&
                        <iframe title="Template preview"
                                className="template-preview"
                                src={ cs.Routes.PreviewTemplate.replace(":id", this.state.previewRecord.id) }>
                        </iframe> }
                </Modal>
            </section>
        )
    }
}

export default Templates
