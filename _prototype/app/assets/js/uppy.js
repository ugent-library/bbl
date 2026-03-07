import Uppy from '@uppy/core';
import DragDrop from '@uppy/drag-drop';
import StatusBar from '@uppy/status-bar';
import AwsS3 from '@uppy/aws-s3';

export default function (rootEl) {
    rootEl.querySelectorAll('[data-uppy]').forEach(el => {
        const presignURL = el.getAttribute('data-uppy-presign-url')

        const uppy = new Uppy({
            debug: true,
            autoProceed: true,
        });

        uppy.use(DragDrop, {
            target: el.querySelector('[data-uppy-drag-drop]'),
        });
        uppy.use(StatusBar, {
            target: el.querySelector('[data-uppy-status]'),
        });
        uppy.use(AwsS3, {
            getUploadParameters(file) {
                return fetch(presignURL, {
                    method: 'post',
                    headers: {
                        'Accept': 'application/json',
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        name: file.name,
                        content_type: file.type,
                    }),
                }).then((res) => {
                    return res.json()
                }).then((data) => {
                    uppy.setFileMeta(file.id, {
                        object_id: data.object_id,
                    });
                    return {
                        method: 'PUT',
                        url: data.url,
                        headers: {
                            'Content-Type': file.type,
                        },
                    }
                })
            },
        });

        // TODO handle res.failed
        uppy.on('complete', (res) => {
            let files = res.successful.map(f => {
                return {
                    object_id: f.meta.object_id,
                    name: f.name,
                    content_type: f.type,
                    size: f.size,
                };
            });

            let inputEl = document.createElement('input');
            inputEl.type = 'hidden';
            inputEl.name = 'files';
            inputEl.value = JSON.stringify(files);
            el.appendChild(inputEl);

            el.dispatchEvent(new Event('files-added'));
        });
    });
}