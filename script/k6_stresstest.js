import http from 'k6/http';
import { check } from 'k6';

export let options = {
    vus: 1000, // 设置并发用户数为 500
    duration: '1m', // 设置测试持续时间为 1 分钟
  };

export default function () {

  let response = http.get('http://127.0.0.1:8080/id?biztag=test');
  // 校验状态码是否为 200
  check(response, {
    'Status is 200': (r) => r.status === 200,
  });

  // 校验返回值是否包含指定的字符串
  check(response, {
    'Response contains "succ"': (r) => r.body.includes('succ'),
  });
}