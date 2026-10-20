"""Regression tests for benchmark accounting, without live services."""
import io
import json
import unittest
from unittest.mock import patch
import gcs_load_test as bench

class Response(io.BytesIO):
    status = 200

class BenchmarkTests(unittest.TestCase):
    def test_upload_business_failure_is_not_success(self):
        with patch.object(bench.urllib.request, 'urlopen', return_value=Response(b'{"code":500,"message":"quota"}')):
            with self.assertRaisesRegex(RuntimeError, 'business_code=500'):
                bench.request_multipart_json('http://localhost/upload', '', {}, b'body')

    def test_range_requires_partial_content(self):
        args = type('Args', (), dict(range_size=10,base_url='http://localhost',file_id='f',token=''))()
        with patch.object(bench, 'request_bytes', return_value=(200,10)):
            with self.assertRaisesRegex(RuntimeError, 'Range mismatch'):
                bench.run_download_range(args,0)

    def test_summary_separates_errors_and_successes(self):
        output=io.StringIO()
        argv=['bench','--mode','identity','--requests','4','--concurrency','1','--monitor','']
        with patch('sys.argv',argv), patch('sys.stdout',output), patch.object(bench,'request_json',side_effect=[({},1),RuntimeError('denied'),({},1),({},1)]):
            self.assertEqual(bench.main(),1)
        data=json.loads(output.getvalue())
        self.assertEqual((data['requests'],data['success'],data['errors']),(4,3,1))
        self.assertEqual(data['error_rate'],0.25)
        self.assertAlmostEqual(data['qps']/data['attempt_qps'],0.75)

if __name__=='__main__': unittest.main()
