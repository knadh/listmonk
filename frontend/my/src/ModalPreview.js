import React from "react"
import { Modal } from "antd"
import * as cs from "./constants"

import { Spin } from "antd"

class ModalPreview extends React.PureComponent {
    state = {
        loading: true
    }

    makeForm(body) {
        let form = document.createElement("form")
        form.method = cs.MethodPost
        form.action = this.props.previewURL
        form.target = "preview-iframe"

        let input = document.createElement("input")
        input.type = "hidden"
        input.name = "body"
        input.value = body
        form.appendChild(input)
        document.body.appendChild(form)
        form.submit()
    }

    render () {
        return (
            <Modal visible={ true } title={ this.props.title }
                className="preview-modal"
                width="90%"
                height={ 900 }
                onCancel={ this.props.onCancel }
                onOk={ this.props.onCancel }>
                <div className="preview-iframe-container">
                    <Spin spinning={ this.state.loading }>
                        <iframe onLoad={() => { this.setState({ loading: false }) }} title={ this.props.title ? this.props.title : "Preview" }
                            name="preview-iframe"
                            id="preview-iframe"
                            className="preview-iframe"
                            ref={(o) => {
                                if(o) {
                                    // When the DOM reference for the iframe is ready,
                                    // see if there's a body to post with the form hack.
                                    if(this.props.body !== undefined
                                        && this.props.body !== null) {
                                        this.makeForm(this.props.body)
                                    }
                                }
                            }}
                            src={ this.props.previewURL ? this.props.previewURL : "about:blank" }>
                        </iframe>
                    </Spin>
                </div>
            </Modal>

        )
    }
}

export default ModalPreview
